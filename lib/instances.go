package autoscaler

type Instances []*Instance

func (is Instances) Managed() Instances {
	instances := Instances{}

L:
	for _, i := range is {
		for _, t := range i.Tags {
			if *t.Key == "ManagedBy" && *t.Value == "spot-autoscaler" {
				instances = append(instances, i)
				continue L
			}
		}
	}

	return instances
}

func (is Instances) Ondemand() Instances {
	instances := Instances{}
	for _, i := range is {
		if i.SpotInstanceRequestId == nil {
			instances = append(instances, i)
		}
	}

	return instances
}
