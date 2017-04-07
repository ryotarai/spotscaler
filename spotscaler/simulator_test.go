package spotscaler

import (
	"testing"
)

func TestSimulateLaunch(t *testing.T) {
	c4large := InstanceVariety{AvailabilityZone: "az-a", InstanceType: "c4.large"}
	c42xlarge := InstanceVariety{AvailabilityZone: "az-a", InstanceType: "c4.2xlarge"}

	s, err := NewSimulator(80.0, 40.0, map[InstanceVariety]int{
		c4large:   10,
		c42xlarge: 40,
	}, 1, 0, 0.6)

	if err != nil {
		t.Fatal(err)
	}

	i := NewInstance("i-1", InstanceVariety{
		InstanceType:     "c4.xlarge",
		AvailabilityZone: "az-a",
	}, 20, LaunchMethodSpot)

	state := &EC2State{
		Instances: Instances{i},
	}

	terminate, keep, launch := s.Simulate(state)

	tests := []struct {
		want interface{}
		got  interface{}
	}{
		{0, len(terminate)},
		{1, len(keep)},
		{4, len(launch)},
		{i, keep[0]},
		{c4large, launch[0].Variety},
		{c42xlarge, launch[1].Variety},
		{c4large, launch[2].Variety},
		{c4large, launch[3].Variety},
		{10, launch[0].Capacity},
		{40, launch[1].Capacity},
		{10, launch[2].Capacity},
		{10, launch[3].Capacity},
	}

	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("want %+v, but got %+v", test.want, test.got)
		}
	}
}

func TestSimulateInitialLaunch(t *testing.T) {
	c4large := InstanceVariety{AvailabilityZone: "az-a", InstanceType: "c4.large"}
	c42xlarge := InstanceVariety{AvailabilityZone: "az-a", InstanceType: "c4.2xlarge"}

	s, err := NewSimulator(80.0, 40.0, map[InstanceVariety]int{
		c4large:   10,
		c42xlarge: 40,
	}, 1, 40, 0.6)

	if err != nil {
		t.Fatal(err)
	}

	state := &EC2State{
		Instances: Instances{},
	}

	terminate, keep, launch := s.Simulate(state)

	tests := []struct {
		want interface{}
		got  interface{}
	}{
		{0, len(terminate)},
		{0, len(keep)},
		{5, len(launch)},
		{c4large, launch[0].Variety},
		{c42xlarge, launch[1].Variety},
		{c4large, launch[2].Variety},
		{c4large, launch[3].Variety},
		{c4large, launch[4].Variety},
		{10, launch[0].Capacity},
		{40, launch[1].Capacity},
		{10, launch[2].Capacity},
		{10, launch[3].Capacity},
		{10, launch[4].Capacity},
	}

	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("want %+v, but got %+v", test.want, test.got)
		}
	}
}

func TestSimulateTerminate(t *testing.T) {
	c4large := InstanceVariety{AvailabilityZone: "az-a", InstanceType: "c4.large"}
	c42xlarge := InstanceVariety{AvailabilityZone: "az-a", InstanceType: "c4.2xlarge"}

	s, err := NewSimulator(10.0, 80.0, map[InstanceVariety]int{
		c4large:   10,
		c42xlarge: 40,
	}, 1, 0, 0.9)

	if err != nil {
		t.Fatal(err)
	}

	i1 := NewInstance("i-1", InstanceVariety{
		InstanceType:     "c4.xlarge",
		AvailabilityZone: "az-a",
	}, 20, LaunchMethodSpot)
	i2 := NewInstance("i-2", InstanceVariety{
		InstanceType:     "c4.2xlarge",
		AvailabilityZone: "az-b",
	}, 40, LaunchMethodSpot)
	i3 := NewInstance("i-3", InstanceVariety{
		InstanceType:     "c4.4xlarge",
		AvailabilityZone: "az-b",
	}, 80, LaunchMethodSpot)

	state := &EC2State{
		Instances: Instances{i1, i2, i3},
	}

	terminate, keep, launch := s.Simulate(state)

	tests := []struct {
		want interface{}
		got  interface{}
	}{
		{1, len(terminate)},
		{2, len(keep)},
		{0, len(launch)},
		{i1, keep[0]},
		{i2, keep[1]},
		{i3, terminate[0]},
	}

	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("want %+v, but got %+v", test.want, test.got)
		}
	}
}
