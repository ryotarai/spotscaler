package ec2

type Instance struct {
	InstanceID       string
	InstanceType     string
	AvailabilityZone string
	Capacity         float64
	Market           string // ondemand, spot
}

type Instances []*Instance

func (is Instances) TotalCapacity() float64 {
	var f float64
	for _, i := range is {
		f += i.Capacity
	}
	return f
}
