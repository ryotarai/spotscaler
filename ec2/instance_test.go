package ec2

import (
	"testing"
)

func TestWithoutTopNVarieties(t *testing.T) {
	is := Instances{
		{
			Tags:    map[string]string{"Capacity": "10"},
			Variety: InstanceVariety{AvailabilityZone: "ap-northeast-1a", InstanceType: "c4.large"},
		},
		{
			Tags:    map[string]string{"Capacity": "20"},
			Variety: InstanceVariety{AvailabilityZone: "ap-northeast-1b", InstanceType: "c4.large"},
		},
		{
			Tags:    map[string]string{"Capacity": "15"},
			Variety: InstanceVariety{AvailabilityZone: "ap-northeast-1a", InstanceType: "m4.large"},
		},
		{
			Tags:    map[string]string{"Capacity": "15"},
			Variety: InstanceVariety{AvailabilityZone: "ap-northeast-1a", InstanceType: "m4.large"},
		},
	}

	filtered, err := is.WithoutTopNVarieties(2)
	if err != nil {
		t.Error(err)
	}
	if len(filtered) != 1 {
		t.Errorf("Got %d instances, but wants %d instances", len(filtered), 1)
	}
	if filtered[0] != is[0] {
		t.Errorf("Got %v, but wants %v", filtered[0], is[0])
	}

	filtered, err = is.WithoutTopNVarieties(1)
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
}
