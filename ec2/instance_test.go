package ec2

import (
	"testing"
)

func TestInWorstCase(t *testing.T) {
	is := Instances{
		{
			Tags:      map[string]string{"Capacity": "40"},
			Variety:   InstanceVariety{AvailabilityZone: "ap-northeast-1b", InstanceType: "m4.large"},
			Lifecycle: "normal",
		},
		{
			Tags:      map[string]string{"Capacity": "10"},
			Variety:   InstanceVariety{AvailabilityZone: "ap-northeast-1a", InstanceType: "c4.large"},
			Lifecycle: "spot",
		},
		{
			Tags:      map[string]string{"Capacity": "20"},
			Variety:   InstanceVariety{AvailabilityZone: "ap-northeast-1b", InstanceType: "c4.large"},
			Lifecycle: "spot",
		},
		{
			Tags:      map[string]string{"Capacity": "15"},
			Variety:   InstanceVariety{AvailabilityZone: "ap-northeast-1a", InstanceType: "m4.large"},
			Lifecycle: "spot",
		},
		{
			Tags:      map[string]string{"Capacity": "15"},
			Variety:   InstanceVariety{AvailabilityZone: "ap-northeast-1a", InstanceType: "m4.large"},
			Lifecycle: "spot",
		},
	}

	filtered, err := is.InWorstCase(2)
	if err != nil {
		t.Error(err)
	}
	if len(filtered) != 2 {
		t.Errorf("Got %d instances, but wants %d instances", len(filtered), 2)
	}
	if filtered[0] != is[0] {
		t.Errorf("Got %v, but wants %v", filtered[0], is[0])
	}
	if filtered[1] != is[1] {
		t.Errorf("Got %v, but wants %v", filtered[1], is[1])
	}

	filtered, err = is.InWorstCase(1)
	if err != nil {
		t.Error(err)
	}
	if len(filtered) != 3 {
		t.Errorf("Got %d instances, but wants %d instances", len(filtered), 3)
	}
	if filtered[0] != is[0] {
		t.Errorf("Got %v, but wants %v", filtered[0], is[0])
	}
	if filtered[1] != is[1] {
		t.Errorf("Got %v, but wants %v", filtered[1], is[1])
	}
	if filtered[2] != is[2] {
		t.Errorf("Got %v, but wants %v", filtered[2], is[2])
	}
}
