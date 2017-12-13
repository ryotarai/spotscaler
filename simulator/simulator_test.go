package simulator

import (
	"testing"

	"github.com/ryotarai/spotscaler/ec2"
	"github.com/stretchr/testify/assert"
)

func TestWorstCase(t *testing.T) {
	s := &Simulator{
		PossibleTermination: 2,
	}

	is := s.WorstCase(ec2.Instances{
		{AvailabilityZone: "az-1", InstanceType: "a", Capacity: 10.0, Market: "spot"},
	})
	assert.Len(t, is, 0)

	all := ec2.Instances{
		{AvailabilityZone: "az-1", InstanceType: "a", Capacity: 10.0, Market: "spot"},
		{AvailabilityZone: "az-1", InstanceType: "a", Capacity: 20.0, Market: "spot"},
		{AvailabilityZone: "az-2", InstanceType: "a", Capacity: 30.0, Market: "spot"},
		{AvailabilityZone: "az-2", InstanceType: "a", Capacity: 40.0, Market: "spot"},
		{AvailabilityZone: "az-1", InstanceType: "b", Capacity: 50.0, Market: "spot"},
		{AvailabilityZone: "az-1", InstanceType: "b", Capacity: 60.0, Market: "spot"},
		{AvailabilityZone: "az-2", InstanceType: "b", Capacity: 70.0, Market: "spot"},
		{AvailabilityZone: "az-2", InstanceType: "b", Capacity: 80.0, Market: "spot"},
		{AvailabilityZone: "az-2", InstanceType: "b", Capacity: 70.0, Market: "ondemand"},
		{AvailabilityZone: "az-2", InstanceType: "b", Capacity: 80.0, Market: "ondemand"},
	}
	is = s.WorstCase(all)
	assert.Len(t, is, 6)
	assert.Equal(t, all[0], is[0])
	assert.Equal(t, all[1], is[1])
	assert.Equal(t, all[2], is[2])
	assert.Equal(t, all[3], is[3])
	assert.Equal(t, all[8], is[4])
	assert.Equal(t, all[9], is[5])
}

func TestDesiredInstances(t *testing.T) {
	s := &Simulator{
		TargetMetric:      75,
		AvailabilityZones: []string{"az-1", "az-2"},
		InstanceTypes:     []string{"a"},
		CapacityByType:    map[string]float64{"a": 10},
	}

	all := ec2.Instances{
		{AvailabilityZone: "az-1", InstanceType: "a", Capacity: 10.0, Market: "spot"},
	}
	desired := s.DesiredInstances(all, 50.0)
	assert.Len(t, desired, 1)

	all = ec2.Instances{
		{AvailabilityZone: "az-1", InstanceType: "a", Capacity: 10.0, Market: "spot"},
		{AvailabilityZone: "az-1", InstanceType: "a", Capacity: 10.0, Market: "spot"},
		{AvailabilityZone: "az-2", InstanceType: "a", Capacity: 10.0, Market: "spot"},
	}
	desired = s.DesiredInstances(all, 50.0)
	assert.Len(t, desired, 2)
	assert.Equal(t, all[0], desired[0])
	assert.Equal(t, all[2], desired[1])

	all = ec2.Instances{
		{AvailabilityZone: "az-1", InstanceType: "a", Capacity: 10.0, Market: "spot"},
		{AvailabilityZone: "az-1", InstanceType: "a", Capacity: 10.0, Market: "spot"},
	}
	desired = s.DesiredInstances(all, 175.0)
	assert.Len(t, desired, 5)
	assert.Equal(t, all[0], desired[0])
	assert.Equal(t, all[1], desired[1])
	assert.Equal(t, &ec2.Instance{
		InstanceType:     "a",
		AvailabilityZone: "az-2",
		Capacity:         10.0,
		Market:           "spot",
	}, desired[2])
	assert.Equal(t, &ec2.Instance{
		InstanceType:     "a",
		AvailabilityZone: "az-2",
		Capacity:         10.0,
		Market:           "spot",
	}, desired[3])
	assert.Equal(t, &ec2.Instance{
		InstanceType:     "a",
		AvailabilityZone: "az-1",
		Capacity:         10.0,
		Market:           "spot",
	}, desired[4])
}
