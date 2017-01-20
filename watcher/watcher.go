package watcher

import (
	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/state"
	"time"
)

type Watcher struct {
	Ui    cli.Ui
	State *state.State
}

func (w *Watcher) Start() error {
	w.Ui.Info("Watch loop started")
	for {
		c := time.After(1 * time.Minute)
		w.Ui.Info("Execution in loop finished, Waiting 1 min")
		<-c
	}
}
