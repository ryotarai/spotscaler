package spotscaler

import (
	"fmt"

	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2State struct {
	Instances Instances
	SpotPrice map[InstanceVariety]float64
}

type EC2 struct {
	SpotProductDescription string
	Varieties              []InstanceVariety
	WorkingFilters         map[string][]string
	sdk                    *ec2.EC2
}

func NewEC2(workingFilters map[string][]string, productDescription string, varieties []InstanceVariety) (*EC2, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	s := ec2.New(sess)
	return &EC2{
		sdk: s,
		SpotProductDescription: productDescription,
		Varieties:              varieties,
		WorkingFilters:         workingFilters,
	}, nil
}

func (e *EC2) GetCurrentState() (*EC2State, error) {
	instances, err := e.getWorkingInstances(e.WorkingFilters)
	if err != nil {
		return nil, err
	}

	spotPrice, err := e.getSpotPrice(e.Varieties)
	if err != nil {
		return nil, err
	}

	return &EC2State{
		Instances: instances,
		SpotPrice: spotPrice,
	}, nil
}

func (e *EC2) getSpotPrice(varieties []InstanceVariety) (map[InstanceVariety]float64, error) {
	result := map[InstanceVariety]float64{}

	typesByAZ := map[string][]string{}
	for _, v := range varieties {
		typesByAZ[v.AvailabilityZone] = append(typesByAZ[v.AvailabilityZone], v.InstanceType)
	}

	for az, types := range typesByAZ {
		input := &ec2.DescribeSpotPriceHistoryInput{
			AvailabilityZone:    aws.String(az),
			InstanceTypes:       aws.StringSlice(types),
			ProductDescriptions: aws.StringSlice([]string{e.SpotProductDescription}),
		}

		priceByType := map[string]float64{}
		var insideErr error
		err := e.sdk.DescribeSpotPriceHistoryPages(input, func(p *ec2.DescribeSpotPriceHistoryOutput, lastPage bool) bool {
			if len(p.SpotPriceHistory) == 0 {
				insideErr = fmt.Errorf("cannot get current spot price in %s", az)
				return false
			}

			for _, h := range p.SpotPriceHistory {
				if priceByType[*h.InstanceType] != 0.0 {
					continue
				}

				price, err := strconv.ParseFloat(*h.SpotPrice, 64)
				if err != nil {
					insideErr = err
					return false
				}

				priceByType[*h.InstanceType] = price

				if len(priceByType) >= len(types) {
					return false
				}
			}
			return true
		})

		if err != nil {
			return nil, err
		}

		if insideErr != nil {
			return nil, insideErr
		}

		for t, price := range priceByType {
			v := InstanceVariety{
				AvailabilityZone: az,
				InstanceType:     t,
			}
			result[v] = price
		}
	}

	return result, nil
}

func (e *EC2) getWorkingInstances(workingFilters map[string][]string) ([]*Instance, error) {
	filters := []*ec2.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("running")},
		},
	}
	for n, vs := range workingFilters {
		filters = append(filters, &ec2.Filter{
			Name:   aws.String(n),
			Values: aws.StringSlice(vs),
		})
	}

	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	output, err := e.sdk.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	is := []*Instance{}
	for _, r := range output.Reservations {
		for _, i := range r.Instances {
			j, err := e.newInstance(i)
			if err != nil {
				return nil, err
			}

			is = append(is, j)
		}
	}

	return is, nil
}

func (e *EC2) newInstance(i *ec2.Instance) (*Instance, error) {
	var method InstanceLaunchMethod
	if i.InstanceLifecycle == nil {
		method = LaunchMethodOndemand
	}
	if *i.InstanceLifecycle == "spot" {
		method = LaunchMethodSpot
	} else {
		return nil, fmt.Errorf("unsupported lifecycle: %s", *i.InstanceLifecycle)
	}

	var capacityTag *ec2.Tag
	for _, t := range i.Tags {
		if *t.Key == "Capacity" {
			capacityTag = t
		}
	}

	if capacityTag == nil {
		return nil, fmt.Errorf("Capacity tag is not found")
	}

	capacity, err := strconv.Atoi(*capacityTag.Value)
	if err != nil {
		return nil, err
	}

	return NewInstance(*i.InstanceId, InstanceVariety{
		AvailabilityZone: *i.Placement.AvailabilityZone,
		InstanceType:     *i.InstanceType,
	}, capacity, method), nil
}
