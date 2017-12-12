package ec2

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

type EC2 struct {
	sdk ec2iface.EC2API

	WorkingFilters map[string][]string
	CapacityTagKey string
}

func New() *EC2 {
	sess := session.Must(session.NewSession())
	return &EC2{
		sdk:            ec2.New(sess),
		WorkingFilters: map[string][]string{},
	}
}

type Instance struct {
	InstanceID       string
	InstanceType     string
	AvailabilityZone string
	Capacity         float64
	Market           string // ondemand, spot
}

func (e *EC2) GetInstances() ([]*Instance, error) {
	filters := []*ec2.Filter{}
	for n, v := range e.WorkingFilters {
		filters = append(filters, &ec2.Filter{Name: aws.String(n), Values: aws.StringSlice(v)})
	}
	output, err := e.sdk.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	instances := []*Instance{}
	for _, r := range output.Reservations {
		for _, i := range r.Instances {
			capStr, err := findTag(i.Tags, e.CapacityTagKey)
			if err != nil {
				return nil, err
			}

			cap, err := strconv.ParseFloat(capStr, 64)
			if err != nil {
				return nil, err
			}

			var market string
			if i.SpotInstanceRequestId == nil {
				market = "ondemand"
			} else {
				market = "spot"
			}

			instances = append(instances, &Instance{
				InstanceID:       *i.InstanceId,
				InstanceType:     *i.InstanceType,
				AvailabilityZone: *i.Placement.AvailabilityZone,
				Capacity:         cap,
				Market:           market,
			})
		}
	}

	return instances, nil
}

func findTag(tags []*ec2.Tag, key string) (string, error) {
	for _, t := range tags {
		if *t.Key == key {
			return *t.Value, nil
		}
	}

	return "", fmt.Errorf("%s tag is not found", key)
}
