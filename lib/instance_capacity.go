package autoscaler

import (
	"fmt"
	"log"
	"math"
	"sort"
)

var capacityTable map[string]float64

type InstanceCapacity map[InstanceVariety]float64

func (c InstanceCapacity) Total() float64 {
	t := 0.0
	for _, cap := range c {
		t += cap
	}
	return t
}

func (c InstanceCapacity) Values() []float64 {
	a := []float64{}
	for _, v := range c {
		a = append(a, v)
	}
	return a
}

func (c InstanceCapacity) Varieties() []InstanceVariety {
	vs := []InstanceVariety{}
	for v, _ := range c {
		vs = append(vs, v)
	}
	return vs
}

func (c InstanceCapacity) Increment() (InstanceCapacity, error) {
	varieties := c.Varieties()
	sort.Sort(SortInstanceVarietiesByCapacity(varieties))

	var leastVariety InstanceVariety
	leastCapacity := math.Inf(1)
	for _, v := range varieties {
		if c[v] < leastCapacity {
			leastCapacity = c[v]
			leastVariety = v
		}
	}

	log.Printf("[TRACE] InstanceCapacity.Increment: adding %v", leastVariety)
	cap, err := leastVariety.Capacity()
	if err != nil {
		return nil, err
	}
	c[leastVariety] += cap

	return c, nil
}

func (c InstanceCapacity) TotalInWorstCase(maxTerminatedVarieties int) float64 {
	values := c.Values()
	sort.Float64s(values)
	total := 0.0
	a := len(values) - maxTerminatedVarieties
	if a < 0 {
		a = 0
	}
	for _, v := range values[:a] {
		total += v
	}

	return total
}

func (cFrom InstanceCapacity) CountDiff(cTo InstanceCapacity) (map[InstanceVariety]int64, error) {
	change := map[InstanceVariety]int64{}

	for v, to := range cTo {
		from := cFrom[v]
		diff := to - from
		if diff > 0 {
			cap, err := v.Capacity()
			if err != nil {
				return nil, err
			}

			count := int64(math.Ceil(diff / cap))
			change[v] = count
		}
	}

	remain := cFrom.Total() - cTo.Total()
	for v, from := range cFrom {
		to := cTo[v]
		diff := from - to
		if diff > 0 {
			cap, err := v.Capacity()
			if err != nil {
				return nil, err
			}

			diff = math.Min(remain, diff)
			count := int64(math.Floor(diff / cap))
			if count > 0 {
				change[v] = count * -1
			}
		}
	}

	return change, nil
}

func DesiredCapacityFromTargetCPUUtil(varieties []InstanceVariety, cpuUtil float64, maxCPUUtil float64, targetCPUUtilDiff float64, ondemandCapacityTotal float64, spotCapacityTotal float64, maxTerminatedVarieties int) (InstanceCapacity, error) {
	var err error
	desiredCapacity := InstanceCapacity{}
	for _, v := range varieties {
		desiredCapacity[v] = 0.0
	}

L:
	for {
		u := cpuUtil * (ondemandCapacityTotal + spotCapacityTotal) / (ondemandCapacityTotal + desiredCapacity.Total())
		uScaleOut := maxCPUUtil *
			(ondemandCapacityTotal + desiredCapacity.TotalInWorstCase(maxTerminatedVarieties)) /
			(ondemandCapacityTotal + desiredCapacity.Total())
		log.Printf("[TRACE] DesiredCapacityFromTargetCPUUtil u: %f, uScaleOut: %f", u, uScaleOut)
		if u < uScaleOut-targetCPUUtilDiff {
			break L
		}

		desiredCapacity, err = desiredCapacity.Increment()
		if err != nil {
			return nil, err
		}
	}

	return desiredCapacity, nil
}

func DesiredCapacityFromTotal(varieties []InstanceVariety, total float64, maxTerminatedVarieties int) (InstanceCapacity, error) {
	var err error
	desiredCapacity := InstanceCapacity{}
	for _, v := range varieties {
		desiredCapacity[v] = 0.0
	}

L:
	for {
		if total <= desiredCapacity.TotalInWorstCase(maxTerminatedVarieties) {
			break L
		}

		desiredCapacity, err = desiredCapacity.Increment()
		if err != nil {
			return nil, err
		}
	}

	return desiredCapacity, nil
}

func SetCapacityTable(c map[string]float64) {
	capacityTable = c
}

func CapacityFromInstanceType(t string) (float64, error) {
	cap, ok := capacityTable[t]
	if !ok {
		return 0.0, fmt.Errorf("Capacity of %s is unknown", t)
	}
	return cap, nil
}
