package scaler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/config"
	"github.com/ryotarai/spotscaler/ec2"
	"github.com/ryotarai/spotscaler/exec"
	"github.com/ryotarai/spotscaler/state"
)

type Scaler struct {
	Ui     cli.Ui
	State  state.State
	EC2    *ec2.Client
	Config *config.Config
}

func (s *Scaler) Start() error {
	s.Ui.Info("Watch loop started")
	for {
		c := time.After(1 * time.Minute)

		err := s.RunOnce()
		if err != nil {
			s.Ui.Error(fmt.Sprint(err))
		}

		s.Ui.Info("Execution in loop finished, waiting 1 min")
		<-c
	}
}

func (s *Scaler) RunOnce() error {
	currentInstances, err := s.EC2.DescribeRunningInstances(s.Config.WorkingInstanceFilters)
	if err != nil {
		return err
	}
	s.Ui.Output(fmt.Sprintf("Current working instances: %s", currentInstances))

	if currentInstances.TotalCapacity() == 0 {
		return fmt.Errorf("Current working instances have no capacity")
	}

	availableVarieties, err := s.ListAvailableVarieties()
	if err != nil {
		return err
	}
	s.Ui.Output(fmt.Sprintf("Available varieties: %+v", availableVarieties))

	worstCaseInstances := currentInstances.InWorstCase(s.Config.MaxTerminatedVarieties)
	s.Ui.Output(fmt.Sprintf("Instances in the worst case: %s", worstCaseInstances))

	scalingOutThreshold := s.Config.ScalingOutThreshold * float64(worstCaseInstances.TotalCapacity()) / float64(currentInstances.TotalCapacity())
	scalingInThreshold := scalingOutThreshold * s.Config.ScalingInThresholdFactor
	s.Ui.Info(fmt.Sprintf("Scaling out threshold: %f", scalingOutThreshold))
	s.Ui.Info(fmt.Sprintf("Scaling in threshold: %f", scalingInThreshold))

	metricValue, err := s.MetricValue()
	if err != nil {
		return err
	}
	s.Ui.Info(fmt.Sprintf("Current metric value: %f", metricValue))

	scaling := false
	if scalingOutThreshold < metricValue {
		s.Ui.Info("Scaling out")
		scaling = true
	} else if metricValue < scalingInThreshold {
		s.Ui.Info("Scaling in")
		scaling = true
	}

	if scaling {
		// desiredInstances, err := s.InstancesAfterScaling(currentInstances, availableVarieties, metricValue)
		// if err != nil {
		// 	return err
		// }
		// s.Ui.Output(fmt.Sprintf("Instances after scaling: %s", desiredInstances))
	}

	return nil
}

func (s *Scaler) InstancesAfterScaling(currentInstances ec2.Instances, availableVarieties []ec2.InstanceVariety, metricValue float64) (ec2.Instances, error) {
	return nil, nil

	// desiredInstancesByVariety := map[ec2.InstanceVariety]ec2.Instances{}
	// desiredInstances := ec2.Instances{}
	// currentSpotInstancesByVariety := map[ec2.InstanceVariety]ec2.Instances{}

	// for _, i := range currentInstances {
	// 	if i.Lifecycle == "normal" {
	// 		desiredInstancesByVariety[i.Variety] = append(desiredInstancesByVariety[i.Variety], i)
	// 		desiredInstances = append(desiredInstances, i)
	// 	} else {
	// 		currentSpotInstancesByVariety[i.Variety] = append(currentSpotInstancesByVariety[i.Variety], i)
	// 	}
	// }

	// intmax := int(^uint(0) >> 1)

	// // current working instances
	// for len(currentSpotInstancesByVariety) > 0 {
	// 	minTSC := intmax
	// 	var variety ec2.InstanceVariety
	// 	for v := range currentSpotInstancesByVariety {
	// 		tsc := desiredInstancesByVariety[v].TotalSpotCapacity()
	// 		if tsc < minTSC {
	// 			minTSC = tsc
	// 			variety = v
	// 		}
	// 	}

	// 	i := currentSpotInstancesByVariety[variety][len(currentSpotInstancesByVariety[variety])-1]
	// 	desiredInstancesByVariety[variety] = append(desiredInstancesByVariety[variety], i)
	// 	desiredInstances = append(desiredInstances, i)

	// 	m := metricValue * float64(currentInstances.TotalCapacity()) / float64(desiredInstances.TotalCapacity())
	// 	outThreshold := s.Config.ScalingOutThreshold * float64(desiredInstances.InWorstCase(s.Config.MaxTerminatedVarieties).TotalCapacity()) / float64(desiredInstances.TotalCapacity())
	// 	inThreshold := outThreshold * s.Config.ScalingInThresholdFactor
	// 	if m < (outThreshold+inThreshold)/2 {
	// 		return desiredInstances, nil
	// 	}

	// 	currentSpotInstancesByVariety[variety] = currentSpotInstancesByVariety[variety][:len(currentSpotInstancesByVariety[variety])-1]
	// 	if len(currentSpotInstancesByVariety[variety]) == 0 {
	// 		delete(currentSpotInstancesByVariety, variety)
	// 	}
	// }

	// // new instances are required
	// for {
	// 	minTSC := intmax
	// 	var variety ec2.InstanceVariety
	// 	for _, v := range availableVarieties {
	// 		tsc := desiredInstancesByVariety[v].TotalSpotCapacity()
	// 		if tsc < minTSC {
	// 			minTSC = tsc
	// 			variety = v
	// 		}
	// 	}

	// 	var instanceType config.InstanceType
	// 	for _, t := range s.Config.InstanceTypes {
	// 		if t.InstanceTypeName == variety.InstanceType {
	// 			instanceType = t
	// 		}
	// 	}

	// 	tags := map[string]string{}
	// 	tags["Capacity"] = fmt.Sprintf("%f", instanceType.Capacity)
	// 	for k, v := range s.Config.InstanceTags {
	// 		tags[k] = v
	// 	}

	// 	i := &ec2.Instance{
	// 		Variety:      variety,
	// 		Tags:         tags,
	// 		Lifecycle:    "spot",
	// 		BiddingPrice: instanceType.BiddingPrice,
	// 	}
	// 	desiredInstances = append(desiredInstances, i)
	// 	desiredInstancesByVariety[variety] = append(desiredInstancesByVariety[variety], i)

	// 	m := metricValue * float64(currentInstances.TotalCapacity()) / float64(desiredInstances.TotalCapacity())
	// 	outThreshold := s.Config.ScalingOutThreshold * float64(desiredInstances.InWorstCase(s.Config.MaxTerminatedVarieties).TotalCapacity()) / float64(desiredInstances.TotalCapacity())
	// 	inThreshold := outThreshold * s.Config.ScalingInThresholdFactor
	// 	if m < (outThreshold+inThreshold)/2 {
	// 		return desiredInstances, nil
	// 	}
	// }
}

func (s *Scaler) MetricValue() (float64, error) {
	e := exec.Executor{
		Command: &s.Config.MetricCommand,
	}
	output, err := e.Run()
	if err != nil {
		return 0.0, err
	}
	v, err := strconv.ParseFloat(strings.TrimSuffix(output, "\n"), 64)
	if err != nil {
		return 0.0, err
	}

	return v, nil
}

func (s *Scaler) ListAvailableVarieties() ([]config.LaunchInstanceVariety, error) {
	vsByAZ := map[string][]config.LaunchInstanceVariety{}
	for _, v := range s.Config.LaunchConfiguration.LaunchInstanceVariety {
		vsByAZ[v.AvailabilityZone] = append(vsByAZ[v.AvailabilityZone], v)
	}

	available := []config.LaunchInstanceVariety{}
	for az, vs := range vsByAZ {
		types := []string{}
		for _, v := range vs {
			types = append(types, v.InstanceType)
		}
		price, err := s.EC2.DescribeCurrentSpotPrice(az, types)
		if err != nil {
			return nil, err
		}
		for _, v := range vs {
			if price[v.InstanceType] <= v.BiddingPrice {
				available = append(available, v)
			}
		}
	}

	return available, nil
}
