package autoscaler

import "fmt"

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

	if ic != jc {
		return ic < jc
	}

	if s[i].Subnet != s[j].Subnet {
		return s[i].Subnet.SubnetID < s[j].Subnet.SubnetID
	}

	if s[i].InstanceType != s[j].InstanceType {
		return s[i].InstanceType < s[j].InstanceType
	}

	panic(fmt.Sprintf("%#v and %#v must be different", s[i], s[j]))
}
