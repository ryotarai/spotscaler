package timer

import (
	"time"

	"github.com/ryotarai/spotscaler/command"
)

type Timer struct {
	Name     string           `yaml:"Name" validate:"required"`
	Duration time.Duration    `yaml:"Duration" validate:"required"`
	After    string           `yaml:"After" validate:"required"`
	Command  *command.Command `yaml:"Command" validate:"required"`
}
