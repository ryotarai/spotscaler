package watcher

import (
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/config"
	"github.com/ryotarai/spotscaler/ec2"
	"github.com/ryotarai/spotscaler/state"
	"strconv"
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
	w.Ui.Output(fmt.Sprintf("Current working instances: %+v", currentInstances))

	price, err := w.DescribeCurrentSpotPrice()
	if err != nil {
		return err
	}
	w.Ui.Output(fmt.Sprintf("Current spot price: %+v", price))

	w.UpdateStatus(currentInstances)

	return nil
}

func (w *Watcher) UpdateStatus(currentInstances ec2.Instances) error {
	currentOndemandCapacity := 0
	for _, i := range currentInstances.FilterByLifecycle("normal") {
		cap, err := strconv.Atoi(i.Tags[w.Config.CapacityTagKey])
		if err != nil {
			return err
		}
		currentOndemandCapacity += cap
	}

	currentSpotCapacity := 0
	for _, i := range currentInstances.FilterByLifecycle("spot") {
		cap, err := strconv.Atoi(i.Tags[w.Config.CapacityTagKey])
		if err != nil {
			return err
		}
		currentSpotCapacity += cap
	}

	w.State.UpdateStatus(&state.Status{
		CurrentOndemandCapacity: currentOndemandCapacity,
		CurrentSpotCapacity:     currentSpotCapacity,
	})
	return nil
}

func (w *Watcher) DescribeCurrentSpotPrice() (map[string]map[string]float64, error) {
	w.Ui.Output("DescribeCurrentSpotPrice")
	azs := []string{}
	for _, s := range w.Config.LaunchConfiguration.Subnets {
		azs = append(azs, s.AvailabilityZone)
	}
	instanceTypes := []string{}
	for _, t := range w.Config.InstanceTypes {
		instanceTypes = append(instanceTypes, t.InstanceTypeName)
	}

	return w.EC2.DescribeCurrentSpotPrice(azs, instanceTypes)
}
