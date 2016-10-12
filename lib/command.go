package autoscaler

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	Command string   `yaml:"Command" validate:"required"`
	Args    []string `yaml:"Args"`
}

func (h Command) RunWithStdin(input string) error {
	log.Printf("[DEBUG] executing %s %v", h.Command, h.Args)

	c := exec.Command(h.Command, h.Args...)
	c.Stdin = strings.NewReader(input)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return err
}

func (h Command) Output(env []string) (string, error) {
	log.Printf("[DEBUG] executing %s %v", h.Command, h.Args)

	env = append(env, os.Environ()...)
	c := exec.Command(h.Command, h.Args...)
	c.Env = env
	b, err := c.Output()
	err = h.wrapError(err)

	return string(b), err
}

func (h Command) wrapError(err error) error {
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("%v: %s", exitError, exitError.Stderr)
		}
	}

	return err
}
