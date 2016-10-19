package autoscaler

type Subnet struct {
	SubnetID         string `yaml:"SubnetID" validate:"required"`
	AvailabilityZone string `yaml:"AvailabilityZone" validate:"required"`
}
