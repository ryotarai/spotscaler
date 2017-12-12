package simulator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ryotarai/spotscaler/ec2"
)

type Simulator struct {
	PossibleTermination int64
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
