package scaler

import (
	"testing"

	"github.com/ryotarai/spotscaler/config"
	"github.com/ryotarai/spotscaler/ec2"
)

func TestInstanceAfterScaling(t *testing.T) {
	currentInstances := ec2.Instances{
		{
			Lifecycle: "normal",
			Tags: map[string]string{
				"Capacity": "10",
			},
		},
	}
	tags := map[string]string{
		"Comment": "spotscaler",
	}
	varieties := []config.LaunchInstanceVariety{
		{
			InstanceType:     "c4.large",
			AvailabilityZone: "ap-northeast-1a",
			Capacity:         10,
			BiddingPrice:     1.0,
		}, {
			InstanceType:     "m4.large",
			AvailabilityZone: "ap-northeast-1a",
			Capacity:         10,
			BiddingPrice:     1.0,
		},
	}

	s := &ScalingStrategy{
		CurrentInstances:         currentInstances,
		MetricValue:              80.0,
		ScalingOutThreshold:      70.0,
		ScalingInThresholdFactor: 0.5,
		MaxTerminatedVarieties:   1,
		LaunchInstanceVarieties:  varieties,
		InstanceTags:             tags,
	}
	instances := s.InstancesAfterScaling()
	t.Log(instances)
}
