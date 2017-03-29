package spotscaler

type Instance struct {
	InstanceID string
}

func NewInstance(instanceID string) *Instance {
	return &Instance{
		InstanceID: instanceID,
	}
}
