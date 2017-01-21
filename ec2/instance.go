package ec2

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Instance struct {
	Tags      map[string]string
	Lifecycle string // normal, spot or scheduled
}

func NewInstanceFromSDK(instance *ec2.Instance) *Instance {
	tags := map[string]string{}
	for _, t := range instance.Tags {
		tags[*t.Key] = *t.Value
	}

	lifecycle := "normal"
	if instance.InstanceLifecycle != nil {
		lifecycle = *instance.InstanceLifecycle
	}

	return &Instance{
		Tags:      tags,
		Lifecycle: lifecycle,
	}
}

type Instances []*Instance

func (is Instances) FilterByLifecycle(lifecycle string) Instances {
	ret := Instances{}
	for _, i := range is {
		if i.Lifecycle == lifecycle {
			ret = append(ret, i)
		}
	}

	return ret
}
