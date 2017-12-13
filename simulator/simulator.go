package simulator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ryotarai/spotscaler/ec2"
)

type Simulator struct {
	PossibleTermination int64
	TargetMetric        float64
	InstanceTypes       []string
	AvailabilityZones   []string
	CapacityByType      map[string]float64
}

func (s *Simulator) WorstCase(instances ec2.Instances) ec2.Instances {
	t := map[string]float64{}
	for _, i := range instances {
		if i.Market == "spot" {
			t[fmt.Sprintf("%s/%s", i.AvailabilityZone, i.InstanceType)] += i.Capacity
		}
	}

	type st struct {
		az           string
		instanceType string
		capacity     float64
	}
	tt := []st{}
	for k, v := range t {
		aztype := strings.SplitN(k, "/", 2)
		tt = append(tt, st{aztype[0], aztype[1], v})
	}

	sort.Slice(tt, func(i, j int) bool {
		return tt[i].capacity > tt[j].capacity
	})

	is := ec2.Instances{}
	if int64(len(tt)) > s.PossibleTermination {
		for _, i := range instances {
			for _, t := range tt[s.PossibleTermination:] {
				if i.Market == "spot" {
					if t.az == i.AvailabilityZone && t.instanceType == i.InstanceType {
						is = append(is, i)
					}
				}
			}
		}
	}
	for _, i := range instances {
		if i.Market != "spot" {
			is = append(is, i)
		}
	}

	return is
}

func (s *Simulator) DesiredInstances(instances ec2.Instances, metric float64) ec2.Instances {
	is := ec2.Instances{}
	spotInstances := ec2.Instances{}
	for _, i := range instances {
		if i.Market == "ondemand" {
			is = append(is, i)
		} else {
			spotInstances = append(spotInstances, i)
		}
	}

	cap := map[string]float64{}
	for _, i := range spotInstances {
		cap[fmt.Sprintf("%s/%s", i.AvailabilityZone, i.InstanceType)] = 0
	}

	for {
		if metric*instances.TotalCapacity()/s.WorstCase(is).TotalCapacity() <= s.TargetMetric {
			return is
		}
		if len(spotInstances) == 0 {
			break
		}

		var minCap float64 = -1
		var instanceIdx int
		for idx, i := range spotInstances {
			c := cap[fmt.Sprintf("%s/%s", i.AvailabilityZone, i.InstanceType)]
			if minCap < 0 || c < minCap {
				minCap = c
				instanceIdx = idx
			} else if c == minCap {
				if i.Capacity < spotInstances[instanceIdx].Capacity {
					instanceIdx = idx
				}
			}
		}

		i := spotInstances[instanceIdx]
		cap[fmt.Sprintf("%s/%s", i.AvailabilityZone, i.InstanceType)] += i.Capacity
		is = append(is, i)
		spotInstances = append(spotInstances[:instanceIdx], spotInstances[instanceIdx+1:]...)
	}

	for {
		if metric*instances.TotalCapacity()/s.WorstCase(is).TotalCapacity() <= s.TargetMetric {
			return is
		}

		var minCap float64 = -1
		var nextAZ, nextType string
		for _, az := range s.AvailabilityZones {
			for _, t := range s.InstanceTypes {
				c := cap[fmt.Sprintf("%s/%s", az, t)]
				if minCap < 0 || c < minCap {
					minCap = c
					nextAZ = az
					nextType = t
				} else if c == minCap {
					if s.CapacityByType[t] < s.CapacityByType[nextType] {
						nextAZ = az
						nextType = t
					}
				}
			}
		}

		is = append(is, &ec2.Instance{
			InstanceType:     nextType,
			AvailabilityZone: nextAZ,
			Capacity:         s.CapacityByType[nextType],
			Market:           "spot",
		})
		cap[fmt.Sprintf("%s/%s", nextAZ, nextType)] += s.CapacityByType[nextType]
	}
}
