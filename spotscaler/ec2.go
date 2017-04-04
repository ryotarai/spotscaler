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
}

type EC2 struct {
	sdk *ec2.EC2
}

func NewEC2() (*EC2, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	s := ec2.New(sess)
	return &EC2{
		sdk: s,
	}, nil
}

func (e *EC2) GetCurrentState(workingFilters map[string][]string) (*EC2State, error) {
	instances, err := e.getWorkingInstances(workingFilters)
	if err != nil {
		return nil, err
	}

	return &EC2State{
		Instances: instances,
	}, nil
}

func (e *EC2) getWorkingInstances(workingFilters map[string][]string) ([]*Instance, error) {
	filters := []*ec2.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("running")},
		},
	}
	for n, vs := range workingFilters {
		values := []*string{}
		for _, v := range vs {
			values = append(values, aws.String(v))
		}

		filters = append(filters, &ec2.Filter{
			Name:   aws.String(n),
			Values: values,
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
