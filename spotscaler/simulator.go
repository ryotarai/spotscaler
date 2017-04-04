package spotscaler

import (
	"fmt"
	"sort"
)

type Simulator struct {
	Metric            float64
	Threshold         float64
	CapacityByVariety map[InstanceVariety]int
	// Number of varieties are terminated at the same time
	PossibleTermination int
}

func NewSimulator(metric, threshold float64, capacityByVariety map[InstanceVariety]int, possibleTermination int) (*Simulator, error) {
	if len(capacityByVariety) <= possibleTermination {
		return nil, fmt.Errorf("num of varieties must be more than possibleTermination value")
	}

	return &Simulator{
		Metric:              metric,
		Threshold:           threshold,
		CapacityByVariety:   capacityByVariety,
		PossibleTermination: possibleTermination,
	}, nil
}

type varietyCapacity struct {
	variety  InstanceVariety
	capacity int
}

// Simulate returns
// running instances to be terminated,
// running instances to be remained and
// instances to be launched
func (s *Simulator) Simulate(state *EC2State) (Instances, Instances, Instances) {
	keep := Instances{}
	launch := Instances{}

	remaining := make(Instances, len(state.Instances))
	copy(remaining, state.Instances)
	sort.Slice(remaining, func(i, j int) bool {
		return remaining[i].Capacity < remaining[j].Capacity
	})

	for len(remaining) > 0 {
		worstCapacity := s.worstCapacity(keep)

		m := s.Metric * float64(state.Instances.TotalCapacity()) / float64(worstCapacity)

		debugf("m: %f\tthreshold: %f\tworst capacity: %d\n", m, s.Threshold, worstCapacity)
		if m <= s.Threshold {
			return remaining, keep, launch
		}

		varieties := []InstanceVariety{}
		for _, i := range remaining {
			varieties = append(varieties, i.Variety)
		}

	L1:
		for _, c := range s.sortedSpotCapacities(keep, varieties) {
			for idx, i := range remaining {
				if c.variety == i.Variety {
					debugf("keep %+v\n", i)
					keep = append(keep, i)
					remaining = append(remaining[:idx], remaining[idx+1:]...)
					break L1
				}
			}
		}
	}

	for {
		all := append(keep, launch...)
		worstCapacity := s.worstCapacity(all)

		m := s.Metric * float64(state.Instances.TotalCapacity()) / float64(worstCapacity)

		debugf("m: %f\tthreshold: %f\tworst capacity: %d\n", m, s.Threshold, worstCapacity)
		if m <= s.Threshold {
			return remaining, keep, launch
		}

		varieties := []InstanceVariety{}
		for v := range s.CapacityByVariety {
			varieties = append(varieties, v)
		}

		sorted := s.sortCapacityByVariety(s.CapacityByVariety)

	L2:
		for _, c := range s.sortedSpotCapacities(all, varieties) {
			for _, d := range sorted {
				if c.variety == d.variety {
					i := NewInstanceToBeLaunched(d.variety, d.capacity, LaunchMethodSpot)
					debugf("launch %+v\n", i)

					launch = append(launch, i)
					break L2
				}
			}
		}
	}
}

func (s *Simulator) worstCapacity(is Instances) int {
	worstCapacity := 0
	spotCapacityByVariety := map[InstanceVariety]int{}
	for _, i := range is {
		switch i.LaunchMethod {
		case LaunchMethodOndemand:
			worstCapacity += i.Capacity
		case LaunchMethodSpot:
			spotCapacityByVariety[i.Variety] += i.Capacity
		}
	}

	spotCapacities := s.sortCapacityByVariety(spotCapacityByVariety)
	l := len(spotCapacities) - s.PossibleTermination
	if l < 0 {
		l = 0
	}
	for _, c := range spotCapacities[:l] {
		worstCapacity += c.capacity
	}

	return worstCapacity
}

func (s *Simulator) sortedSpotCapacities(is Instances, varieties []InstanceVariety) []varietyCapacity {
	spotCapacityByVariety := map[InstanceVariety]int{}
	for _, v := range varieties {
		spotCapacityByVariety[v] = 0
	}

	for _, i := range is {
		_, ok := spotCapacityByVariety[i.Variety]
		if !ok {
			continue
		}

		if i.LaunchMethod == LaunchMethodSpot {
			spotCapacityByVariety[i.Variety] += i.Capacity
		}
	}

	spotCapacities := s.sortCapacityByVariety(spotCapacityByVariety)

	return spotCapacities
}

func (s *Simulator) sortCapacityByVariety(capacity map[InstanceVariety]int) []varietyCapacity {
	ret := []varietyCapacity{}
	for v, c := range capacity {
		ret = append(ret, varietyCapacity{v, c})
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].capacity < ret[j].capacity
	})

	return ret
}
