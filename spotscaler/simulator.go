package spotscaler

import (
	"fmt"
	"sort"
)

type Simulator struct {
	Metric            float64
	Threshold         float64
	CapacityByVariety map[InstanceVariety]int
	InitialCapacity   int
	ScalingInFactor   float64
	MinimumCapacity   int
	MaximumCapacity   int
	// Number of varieties are terminated at the same time
	PossibleTermination int
}

func NewSimulator(metric, threshold float64, capacityByVariety map[InstanceVariety]int, possibleTermination int, initialCapacity int, scalingInFactor float64, minimumCapacity int, maximumCapacity int) (*Simulator, error) {
	if len(capacityByVariety) <= possibleTermination {
		return nil, fmt.Errorf("num of varieties must be more than possibleTermination value")
	}

	if scalingInFactor >= 1 {
		return nil, fmt.Errorf("scalingInFactor must be less than 1 (but got %f)", scalingInFactor)
	}

	return &Simulator{
		Metric:              metric,
		Threshold:           threshold,
		CapacityByVariety:   capacityByVariety,
		PossibleTermination: possibleTermination,
		InitialCapacity:     initialCapacity,
		ScalingInFactor:     scalingInFactor,
		MinimumCapacity:     minimumCapacity,
		MaximumCapacity:     maximumCapacity,
	}, nil
}

// Simulate returns
// running instances to be terminated,
// running instances to be remained and
// instances to be launched
func (s *Simulator) Simulate(state *EC2State) (Instances, Instances, Instances, error) {
	keep := Instances{}
	launch := Instances{}

	remaining := make(Instances, len(state.Instances))
	copy(remaining, state.Instances)
	sort.Slice(remaining, func(i, j int) bool {
		return remaining[i].Capacity < remaining[j].Capacity
	})

	targetMetric := s.Threshold * (s.ScalingInFactor + (1-s.ScalingInFactor)/2)
	debugf("target metric: %f\n", targetMetric)

	for len(remaining) > 0 {
		worstCapacity := s.worstCapacity(keep)
		debugf("worst capacity: %d\n", worstCapacity)

		if keep.TotalCapacity() > s.MaximumCapacity {
			return nil, nil, nil, fmt.Errorf("exceeded maximum capacity (%d)", s.MaximumCapacity)
		}

		m := s.Metric * float64(state.Instances.TotalCapacity()) / float64(worstCapacity)

		debugf("m: %f\n", m)
		if m <= targetMetric && worstCapacity >= s.MinimumCapacity {
			return remaining, keep, launch, nil
		}

		varieties := []InstanceVariety{}
		for _, i := range remaining {
			varieties = append(varieties, i.Variety)
		}

		i := s.nextInstance(keep, remaining)
		debugf("keep %+v\n", i)
		keep = append(keep, i)
		for j, k := range remaining {
			if i == k {
				remaining = append(remaining[:j], remaining[j+1:]...)
			}
		}
		debugf("---\n")
	}

	for {
		all := append(keep, launch...)
		worstCapacity := s.worstCapacity(all)
		debugf("worst capacity: %d\n", worstCapacity)

		if all.TotalCapacity() > s.MaximumCapacity {
			return nil, nil, nil, fmt.Errorf("exceeded maximum capacity (%d)", s.MaximumCapacity)
		}

		if len(state.Instances) == 0 {
			debugf("initial capacity: %d\n", s.InitialCapacity)
			if worstCapacity >= s.InitialCapacity && worstCapacity >= s.MinimumCapacity {
				return remaining, keep, launch, nil
			}
		} else {
			m := s.Metric * float64(state.Instances.TotalCapacity()) / float64(worstCapacity)

			debugf("m: %f\n", m)
			if m <= targetMetric && worstCapacity >= s.MinimumCapacity {
				return remaining, keep, launch, nil
			}
		}

		candidates := Instances{}
		for v, c := range s.CapacityByVariety {
			candidates = append(candidates, NewInstanceToBeLaunched(v, c, LaunchMethodSpot))
		}
		i := s.nextInstance(all, candidates)
		debugf("launch %+v\n", i)
		launch = append(launch, i)

		debugf("---\n")
	}
}

func (s *Simulator) nextInstance(current Instances, candidates Instances) *Instance {
	type st struct {
		totalCapacity int
		instance      *Instance
	}

	total := map[InstanceVariety]int{}
	for _, i := range current {
		if i.LaunchMethod == LaunchMethodSpot {
			total[i.Variety] += i.Capacity
		}
	}

	slice := []st{}
	for _, i := range candidates {
		slice = append(slice, st{
			instance:      i,
			totalCapacity: total[i.Variety],
		})
	}

	sort.Slice(slice, func(i, j int) bool {
		a := slice[i]
		b := slice[j]
		if a.totalCapacity == b.totalCapacity {
			return a.instance.Capacity < b.instance.Capacity
		}
		return a.totalCapacity < b.totalCapacity
	})

	return slice[0].instance
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

	type st struct {
		variety  InstanceVariety
		capacity int
	}

	spotCapacities := []st{}
	for v, c := range spotCapacityByVariety {
		spotCapacities = append(spotCapacities, st{v, c})
	}

	sort.Slice(spotCapacities, func(i, j int) bool {
		return spotCapacities[i].capacity < spotCapacities[j].capacity
	})

	l := len(spotCapacities) - s.PossibleTermination
	if l < 0 {
		l = 0
	}
	for _, c := range spotCapacities[:l] {
		worstCapacity += c.capacity
	}

	return worstCapacity
}
