package autoscaler

type Instances []*Instance

func (is Instances) ManagedBy(managedBy string) Instances {
	instances := Instances{}

L:
	for _, i := range is {
		for _, t := range i.Tags {
			if *t.Key == "ManagedBy" && *t.Value == managedBy {
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

func (is Instances) Spot() Instances {
	instances := Instances{}
	for _, i := range is {
		if i.SpotInstanceRequestId != nil {
			instances = append(instances, i)
		}
	}

	return instances
}

func (is Instances) Capacity() (InstanceCapacity, error) {
	c := InstanceCapacity{}
	for _, i := range is {
		cap, err := CapacityFromInstanceType(*i.InstanceType)
		if err != nil {
			return nil, err
		}
		c[i.Variety()] += cap
	}
	return c, nil
}
