package watcher

import (
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/config"
	"github.com/ryotarai/spotscaler/ec2"
	"github.com/ryotarai/spotscaler/state"
	"time"
)

type Watcher struct {
	Ui     cli.Ui
	State  *state.State
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

	return nil
}
