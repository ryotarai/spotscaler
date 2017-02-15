package scaler

import (
	"fmt"

	"github.com/ryotarai/spotscaler/config"
	"github.com/ryotarai/spotscaler/ec2"
)

type ScalingStrategy struct {
	CurrentInstances         ec2.Instances
	MetricValue              float64
	ScalingOutThreshold      float64
	ScalingInThresholdFactor float64
	MaxTerminatedVarieties   int
	LaunchInstanceVarieties  []config.LaunchInstanceVariety
	InstanceTags             map[string]string
}

func (s *ScalingStrategy) InstancesAfterScaling() ec2.Instances {
	desiredInstances := ec2.Instances{}
	remainingInstances := map[ec2.InstanceVariety]ec2.Instances{}

	// initialize remainingInstances
	for _, i := range s.CurrentInstances {
		if i.Lifecycle == "normal" {
			// ondemand
			desiredInstances = append(desiredInstances, i)
		} else {
			remainingInstances[i.Variety] = append(remainingInstances[i.Variety], i)
		}
	}

	for {
		// check condition
		m := s.MetricValue * float64(s.CurrentInstances.TotalCapacity()) / float64(desiredInstances.TotalCapacity())
		outThreshold := s.ScalingOutThreshold * float64(desiredInstances.InWorstCase(s.MaxTerminatedVarieties).TotalCapacity()) / float64(desiredInstances.TotalCapacity())
		inThreshold := outThreshold * s.ScalingInThresholdFactor

		if m < (outThreshold+inThreshold)/2 {
			// done
			break
		}

		if len(remainingInstances) > 0 {
			// current running instances remain
			minCap := -1
			var suitableVariety ec2.InstanceVariety
			spotDesiredInstanceCapacity := desiredInstances.FilterByLifecycle("spot").TotalCapacityByVariety()
			for v := range remainingInstances {
				cap := spotDesiredInstanceCapacity[v]
				if minCap == -1 || cap < minCap {
					minCap = cap
					suitableVariety = v
				}
			}

			// pick up instances with suitableVariety
			is := remainingInstances[suitableVariety]
			desiredInstances = append(desiredInstances, is[len(is)-1])
			remainingInstances[suitableVariety] = is[:len(is)-1]

			if len(remainingInstances[suitableVariety]) == 0 {
				delete(remainingInstances, suitableVariety)
			}
		} else {
			// launch new instance
			spotDesiredInstanceCapacity := desiredInstances.FilterByLifecycle("spot").TotalCapacityByVariety()

			minCap := -1
			var variety config.LaunchInstanceVariety
			for _, v := range s.LaunchInstanceVarieties {
				w := ec2.InstanceVariety{
					AvailabilityZone: v.AvailabilityZone,
					InstanceType:     v.InstanceType,
				}

				cap := spotDesiredInstanceCapacity[w]
				if minCap == -1 || cap < minCap {
					minCap = cap
					variety = v
				}
			}

			tags := map[string]string{}
			tags["Capacity"] = fmt.Sprintf("%d", variety.Capacity)
			for k, v := range s.InstanceTags {
				tags[k] = v
			}

			i := &ec2.Instance{
				Variety: ec2.InstanceVariety{
					AvailabilityZone: variety.AvailabilityZone,
					InstanceType:     variety.InstanceType,
				},
				Tags:         tags,
				Lifecycle:    "spot",
				BiddingPrice: variety.BiddingPrice,
				SubnetID:     variety.SubnetID,
			}

			desiredInstances = append(desiredInstances, i)
		}
	}
	return desiredInstances
}
