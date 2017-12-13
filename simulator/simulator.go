package simulator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ryotarai/spotscaler/ec2"
	"github.com/sirupsen/logrus"
)

type Simulator struct {
	Logger *logrus.Logger

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

func (s *Simulator) DesiredInstancesFromCapacity(instances ec2.Instances, capacity float64) ec2.Instances {
	return s.desiredInstances(instances, func(is ec2.Instances) bool {
		return is.TotalCapacity() >= capacity
	})
}

func (s *Simulator) DesiredInstancesFromMetric(instances ec2.Instances, metric float64) ec2.Instances {
	return s.desiredInstances(instances, func(is ec2.Instances) bool {
		return metric*instances.TotalCapacity()/s.WorstCase(is).TotalCapacity() <= s.TargetMetric
	})
}

func (s *Simulator) desiredInstances(instances ec2.Instances, satisfy func(is ec2.Instances) bool) ec2.Instances {
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
		if satisfy(is) {
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
		s.Logger.Debugf("KEEP %#v", i)
		spotInstances = append(spotInstances[:instanceIdx], spotInstances[instanceIdx+1:]...)
	}

	for {
		if satisfy(is) {
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

		i := &ec2.Instance{
			InstanceType:     nextType,
			AvailabilityZone: nextAZ,
			Capacity:         s.CapacityByType[nextType],
			Market:           "spot",
		}
		is = append(is, i)
		s.Logger.Debugf("LAUNCH %#v", i)
		cap[fmt.Sprintf("%s/%s", nextAZ, nextType)] += s.CapacityByType[nextType]
	}
}
