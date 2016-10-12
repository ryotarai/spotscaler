package autoscaler

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Tag struct {
	Key   string `yaml:"Key" validate:"required"`
	Value string `yaml:"Value" validate:"required"`
}

type EC2Tags []EC2Tag

func (ts EC2Tags) SDK() []*ec2.Tag {
	ret := []*ec2.Tag{}
	for _, t := range ts {
		ret = append(ret, &ec2.Tag{
			Key:   aws.String(t.Key),
			Value: aws.String(t.Value),
		})
	}
	return ret
}
