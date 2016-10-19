package autoscaler

import (
	"fmt"
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
