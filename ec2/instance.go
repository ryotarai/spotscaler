package ec2

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Instance struct {
	Tags map[string]string
}

type Instances []*Instance

func NewInstanceFromSDK(instance *ec2.Instance) *Instance {
	tags := map[string]string{}
	for _, t := range instance.Tags {
		tags[*t.Key] = *t.Value
	}

	return &Instance{
		Tags: tags,
	}
}
