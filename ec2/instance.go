package ec2

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sort"
	"strconv"
)

type Instance struct {
	ID        string
	Tags      map[string]string
	Lifecycle string // normal, spot or scheduled
	Variety   InstanceVariety
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
	}
}

func (i *Instance) Capacity() (int, error) {
	cap, ok := i.Tags["Capacity"]
	if !ok {
		return 0, fmt.Errorf("%v does not have Capacity tag", i)
	}
	capInt, err := strconv.Atoi(cap)
	if err != nil {
		return 0, err
	}

	return capInt, nil
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

func (is Instances) InWorstCase(numVarieties int) (Instances, error) {
	spotCapacityByVariety := map[InstanceVariety]int{}
	for _, i := range is {
		if i.Lifecycle != "spot" {
			continue
		}
		cap, err := i.Capacity()
		if err != nil {
			return nil, err
		}
		spotCapacityByVariety[i.Variety] += cap
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

	return result, nil
}

type CapacityByVariety struct {
	Capacity int
	Variety  InstanceVariety
}

type CapacitiesByVariety []CapacityByVariety

func (a CapacitiesByVariety) Len() int           { return len(a) }
func (a CapacitiesByVariety) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a CapacitiesByVariety) Less(i, j int) bool { return a[i].Capacity < a[j].Capacity }
