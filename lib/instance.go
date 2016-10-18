package autoscaler

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Instance struct {
	ec2.Instance
}

func NewInstanceFromSDK(i *ec2.Instance) *Instance {
	return &Instance{Instance: *i}
}

func (i *Instance) Variety() InstanceVariety {
	return InstanceVariety{
		InstanceType:     *i.InstanceType,
		LaunchMethod:     i.LaunchMethod(),
		SubnetID:         *i.SubnetId,
		AvailabilityZone: *i.Placement.AvailabilityZone,
	}
}

func (i *Instance) LaunchMethod() string {
	if i.SpotInstanceRequestId == nil {
		return "ondemand"
	}
	return "spot"
}
