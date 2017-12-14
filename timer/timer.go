package timer

import (
	"time"

	"github.com/ryotarai/spotscaler/command"
)

type Timer struct {
	Duration time.Duration
	After    string
	Command  *command.Command
}
