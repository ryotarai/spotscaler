package watcher

import (
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/config"
	"github.com/ryotarai/spotscaler/ec2"
	"github.com/ryotarai/spotscaler/exec"
	"github.com/ryotarai/spotscaler/state"
	"strconv"
	"strings"
	"time"
)

type Watcher struct {
	Ui     cli.Ui
	State  state.State
	EC2    *ec2.Client
	Config *config.Config
}

func (w *Watcher) Start() error {
	w.Ui.Info("Watch loop started")
	for {
		c := time.After(1 * time.Minute)

		err := w.RunOnce()
		if err != nil {
			w.Ui.Error(fmt.Sprint(err))
		}

		w.Ui.Info("Execution in loop finished, waiting 1 min")
		<-c
	}
}

func (w *Watcher) RunOnce() error {
	currentInstances, err := w.EC2.DescribeRunningInstances(w.Config.WorkingInstanceFilters)
	if err != nil {
		return err
	}
	w.Ui.Output(fmt.Sprintf("Current working instances: %s", currentInstances))

	if currentInstances.TotalCapacity() == 0 {
		return fmt.Errorf("Current working instances have no capacity")
	}

	availableVarieties, err := w.ListAvailableInstanceVarieties()
	if err != nil {
		return err
	}
	w.Ui.Output(fmt.Sprintf("Available varieties: %+v", availableVarieties))

	worstCaseInstances, err := currentInstances.InWorstCase(w.Config.MaxTerminatedVarieties)
	if err != nil {
		return err
	}
	w.Ui.Output(fmt.Sprintf("Instances in the worst case: %s", worstCaseInstances))

	scalingOutThreshold := w.Config.ScalingOutThreshold * float64(worstCaseInstances.TotalCapacity()) / float64(currentInstances.TotalCapacity())
	scalingInThreshold := scalingOutThreshold * w.Config.ScalingInThresholdFactor
	w.Ui.Info(fmt.Sprintf("Scaling out threshold: %f", scalingOutThreshold))
	w.Ui.Info(fmt.Sprintf("Scaling in threshold: %f", scalingInThreshold))

	metricValue, err := w.MetricValue()
	if err != nil {
		return err
	}
	w.Ui.Info(fmt.Sprintf("Current metric value: %f", metricValue))

	return nil
}

func (w *Watcher) MetricValue() (float64, error) {
	e := exec.Executor{
		Command: &w.Config.MetricCommand,
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

func (w *Watcher) ListAvailableInstanceVarieties() ([]ec2.InstanceVariety, error) {
	w.Ui.Output("DescribeCurrentSpotPrice")
	azs := []string{}
	for _, s := range w.Config.LaunchConfiguration.Subnets {
		azs = append(azs, s.AvailabilityZone)
	}
	instanceTypes := []string{}
	for _, t := range w.Config.InstanceTypes {
		instanceTypes = append(instanceTypes, t.InstanceTypeName)
	}

	price, err := w.EC2.DescribeCurrentSpotPrice(azs, instanceTypes)
	if err != nil {
		return nil, err
	}

	available := []ec2.InstanceVariety{}
	for _, s := range w.Config.LaunchConfiguration.Subnets {
		for _, t := range w.Config.InstanceTypes {
			v := ec2.InstanceVariety{
				AvailabilityZone: s.AvailabilityZone,
				InstanceType:     t.InstanceTypeName,
			}
			if price[v] < t.BiddingPrice {
				available = append(available, v)
			}
		}
	}

	return available, nil
}
