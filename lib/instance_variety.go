package autoscaler

type InstanceVariety struct {
	InstanceType     string `yaml:"InstanceType" validate:"required"`
	LaunchMethod     string `yaml:"LaunchMethod" validate:"required"`
	SubnetID         string `yaml:"SubnetID" validate:"required"`
	AvailabilityZone string `yaml:"AvailabilityZone" validate:"required"`
}

func (v InstanceVariety) Capacity() (float64, error) {
	return CapacityFromInstanceType(v.InstanceType)
}
