package autoscaler

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Filter struct {
	Name   string   `yaml:"Name" validate:"required"`
	Values []string `yaml:"Values" validate:"required"`
}

type EC2Filters []EC2Filter

func (fs EC2Filters) SDK() []*ec2.Filter {
	ret := []*ec2.Filter{}
	for _, f := range fs {
		values := []*string{}
		for _, v := range f.Values {
			values = append(values, aws.String(v))
		}

		ret = append(ret, &ec2.Filter{
			Name:   aws.String(f.Name),
			Values: values,
		})
	}
	return ret
}
