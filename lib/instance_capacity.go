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
	sort.Sort(sort.Reverse(SortInstanceVarietiesByCapacity(varieties)))

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
