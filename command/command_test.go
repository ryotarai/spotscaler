package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandGetFloat(t *testing.T) {
	c := &Command{
		Path: "/bin/echo",
		Args: []string{"1.25"},
	}

	f, err := c.GetFloat()
	assert.Nil(t, err)
	assert.Equal(t, 1.25, f)
}

func TestCommandGetFloatError(t *testing.T) {
	c := &Command{
		Path: "/bin/bash",
		Args: []string{"-c", "echo ERROR >&2; exit 1"},
	}

	_, err := c.GetFloat()
	assert.Equal(t, 1, GetExitStatusFromError(err))
	assert.Equal(t, "ERROR\n", GetStderrFromError(err))
}
