package exec

import (
	"github.com/ryotarai/spotscaler/config"
	"os/exec"
)

type Executor struct {
	Command *config.Command
}

func (e *Executor) Run() (string, error) {
	c := exec.Command(e.Command.Command, e.Command.Args...)
	b, err := c.Output()
	if err != nil {
		return "", err
	}
	return string(b), nil
}
