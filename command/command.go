package command

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type Command struct {
	Path string   `yaml:"Path" validate:"required"`
	Args []string `yaml:"Args"`
}

func (c *Command) GetFloat() (float64, error) {
	cmd := exec.Command(c.Path, c.Args...)

	b, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	s := string(b)
	s = strings.TrimRight(s, "\n")

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return f, nil
}

func (c *Command) GetString() (string, error) {
	cmd := exec.Command(c.Path, c.Args...)

	b, err := cmd.Output()
	if err != nil {
		return "", err
	}

	s := string(b)
	return strings.TrimRight(s, "\n"), nil
}

func GetExitStatusFromError(err error) int {
	if e, ok := err.(*exec.ExitError); ok {
		if s, ok := e.Sys().(syscall.WaitStatus); ok {
			return s.ExitStatus()
		}
		panic(errors.New("not implemented syscall.WaitStatus"))
	}
	panic(errors.New("not exec.ExitError"))
}

func GetStderrFromError(err error) string {
	if e, ok := err.(*exec.ExitError); ok {
		return string(e.Stderr)
	}
	panic(errors.New("not exec.ExitError"))
}
