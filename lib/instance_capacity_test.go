package autoscaler

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCapacityFromInstanceType(t *testing.T) {
	_, err := CapacityFromInstanceType("unknown")
	assert.Error(t, err)
}

func TestDiffCount(t *testing.T) {
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

	expected := map[InstanceVariety]int64{
		v1: 1,
		v2: -1,
	}

	count, err := from.CountDiff(to)
	assert.NoError(t, err)
	assert.Equal(t, expected, count)

	from = InstanceCapacity{
		v1: 20.0,
	}
	to = InstanceCapacity{
		v2: 10.0,
	}

	expected = map[InstanceVariety]int64{
		v1: -1,
		v2: 1,
	}

	count, err = from.CountDiff(to)
	assert.NoError(t, err)
	assert.Equal(t, expected, count)
}

func TestDesiredCapacityFromTotal(t *testing.T) {
	SetCapacityTable(map[string]float64{
		"c4.large": 10.0,
		"m4.large": 20.0,
		"r3.large": 30.0,
	})

	subnet := Subnet{
		SubnetID:         "subnet-abc",
		AvailabilityZone: "ap-northeast-1a",
	}
	varieties := []InstanceVariety{
		{
			InstanceType: "c4.large",
			Subnet:       subnet,
		},
		{
			InstanceType: "m4.large",
			Subnet:       subnet,
		},
		{
			InstanceType: "r3.large",
			Subnet:       subnet,
		},
	}
	actual, err := DesiredCapacityFromTotal(varieties, 100, 2)
	assert.NoError(t, err)
	assert.Equal(t, InstanceCapacity{
		varieties[0]: 100,
		varieties[1]: 100,
		varieties[2]: 120,
	}, actual)
}
