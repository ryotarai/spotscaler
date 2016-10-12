package autoscaler

import (
	"math"
)

type InstanceCapacityChange struct {
	from InstanceCapacity
	to   InstanceCapacity
}

func NewInstanceCapacityChange(from InstanceCapacity, to InstanceCapacity) *InstanceCapacityChange {
	return &InstanceCapacityChange{
		from: from,
		to:   to,
	}
}

func (c *InstanceCapacityChange) Count() (map[InstanceVariety]int64, error) {
	change := map[InstanceVariety]int64{}

	for v, to := range c.to {
		from := c.from[v]
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

	remain := c.from.Total() - c.to.Total()
	for v, from := range c.from {
		to := c.to[v]
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
