package spotscaler

import (
	"os/exec"
	"strings"
)

type Command struct {
	Name string   `yaml:"name" validate:"required"`
	Args []string `yaml:"args" validate:"required"`
}

func (c *Command) Run() (string, error) {
	cmd := exec.Command(c.Name, c.Args...)
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}
