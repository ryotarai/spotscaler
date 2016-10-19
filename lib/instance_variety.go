package autoscaler

type InstanceVariety struct {
	InstanceType string
	Subnet       Subnet
}

func (v InstanceVariety) Capacity() (float64, error) {
	return CapacityFromInstanceType(v.InstanceType)
}

type SortInstanceVarietiesByCapacity []InstanceVariety

func (s SortInstanceVarietiesByCapacity) Len() int {
	return len(s)
}
func (s SortInstanceVarietiesByCapacity) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SortInstanceVarietiesByCapacity) Less(i, j int) bool {
	ic, err := s[i].Capacity()
	if err != nil {
		panic(err)
	}

	jc, err := s[j].Capacity()
	if err != nil {
		panic(err)
	}

	return ic < jc
}
