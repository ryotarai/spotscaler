package ec2

import (
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go/service/ec2"
)

type Instance struct {
	ID           string
	Tags         map[string]string
	Lifecycle    string // normal, spot or scheduled
	Variety      InstanceVariety
	BiddingPrice float64
	SubnetID     string
}

func NewInstanceFromSDK(instance *ec2.Instance) *Instance {
	tags := map[string]string{}
	for _, t := range instance.Tags {
		tags[*t.Key] = *t.Value
	}

	lifecycle := "normal"
	if instance.InstanceLifecycle != nil {
		lifecycle = *instance.InstanceLifecycle
	}

	variety := InstanceVariety{
		InstanceType:     *instance.InstanceType,
		AvailabilityZone: *instance.Placement.AvailabilityZone,
	}

	return &Instance{
		ID:        *instance.InstanceId,
		Tags:      tags,
		Lifecycle: lifecycle,
		Variety:   variety,
		SubnetID:  *instance.SubnetId,
	}
}

func (i *Instance) Capacity() int {
	cap, ok := i.Tags["Capacity"]
	if ok {
		capInt, err := strconv.Atoi(cap)
		if err == nil {
			return capInt
		}
	}

	return 0
}

type Instances []*Instance

func (is Instances) FilterByLifecycle(lifecycle string) Instances {
	ret := Instances{}
	for _, i := range is {
		if i.Lifecycle == lifecycle {
			ret = append(ret, i)
		}
	}

	return ret
}

func (is Instances) TotalCapacityByVariety() map[InstanceVariety]int {
	total := map[InstanceVariety]int{}
	for _, i := range is {
		total[i.Variety] += i.Capacity()
	}
	return total
}

func (is Instances) InWorstCase(numVarieties int) Instances {
	spotCapacityByVariety := map[InstanceVariety]int{}
	for _, i := range is {
		if i.Lifecycle != "spot" {
			continue
		}
		spotCapacityByVariety[i.Variety] += i.Capacity()
	}

	capacities := CapacitiesByVariety{}
	for v, c := range spotCapacityByVariety {
		capacities = append(capacities, CapacityByVariety{
			Capacity: c,
			Variety:  v,
		})
	}
	sort.Sort(capacities)

	result := Instances{}
	len := len(capacities) - numVarieties
	if len < 0 {
		len = 0
	}

	for _, i := range is {
		if i.Lifecycle == "normal" {
			result = append(result, i)
			continue
		}

		for _, c := range capacities[0:len] {
			if i.Variety == c.Variety {
				result = append(result, i)
			}
		}
	}

	return result
}

func (is Instances) TotalCapacity() int {
	total := 0
	for _, i := range is {
		total += i.Capacity()
	}
	return total
}

func (is Instances) TotalSpotCapacity() int {
	total := 0
	for _, i := range is {
		if i.Lifecycle == "spot" {
			total += i.Capacity()
		}
	}
	return total
}

type CapacityByVariety struct {
	Capacity int
	Variety  InstanceVariety
}

type CapacitiesByVariety []CapacityByVariety

func (a CapacitiesByVariety) Len() int           { return len(a) }
func (a CapacitiesByVariety) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a CapacitiesByVariety) Less(i, j int) bool { return a[i].Capacity < a[j].Capacity }
