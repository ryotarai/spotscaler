package spotscaler

type InstanceLaunchMethod int

const (
	LaunchMethodOndemand InstanceLaunchMethod = iota
	LaunchMethodSpot
)

type InstanceVariety struct {
	InstanceType     string
	AvailabilityZone string
}

type Instance struct {
	InstanceID   string
	Variety      InstanceVariety
	Capacity     int
	LaunchMethod InstanceLaunchMethod
}

func NewInstance(instanceID string, v InstanceVariety, c int, m InstanceLaunchMethod) *Instance {
	return &Instance{
		InstanceID:   instanceID,
		Variety:      v,
		Capacity:     c,
		LaunchMethod: m,
	}
}

func NewInstanceToBeLaunched(v InstanceVariety, c int, m InstanceLaunchMethod) *Instance {
	return &Instance{
		Variety:      v,
		Capacity:     c,
		LaunchMethod: m,
	}
}

type Instances []*Instance

func (is Instances) TotalCapacity() int {
	total := 0
	for _, i := range is {
		total += i.Capacity
	}
	return total
}
