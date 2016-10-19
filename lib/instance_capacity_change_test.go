package autoscaler

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCount(t *testing.T) {
	SetCapacityTable(map[string]float64{
		"t1": 10.0,
		"t2": 10.0,
	})

	v1 := InstanceVariety{
		InstanceType: "t1",
		Subnet: Subnet{
			SubnetID:         "subnet-abc",
			AvailabilityZone: "ap-northeast-1a",
		},
	}
	v2 := InstanceVariety{
		InstanceType: "t2",
		Subnet: Subnet{
			SubnetID:         "subnet-abc",
			AvailabilityZone: "ap-northeast-1a",
		},
	}

	from := InstanceCapacity{
		v1: 10.0,
		v2: 30.0,
	}
	to := InstanceCapacity{
		v1: 20.0,
		v2: 10.0,
	}

	change := NewInstanceCapacityChange(from, to)
	expected := map[InstanceVariety]int64{
		v1: 1,
		v2: -1,
	}

	count, err := change.Count()
	assert.NoError(t, err)
	assert.Equal(t, expected, count)

	from = InstanceCapacity{
		v1: 20.0,
	}
	to = InstanceCapacity{
		v2: 10.0,
	}

	change = NewInstanceCapacityChange(from, to)
	expected = map[InstanceVariety]int64{
		v1: -1,
		v2: 1,
	}

	count, err = change.Count()
	assert.NoError(t, err)
	assert.Equal(t, expected, count)
}
