package cli

import (
	"github.com/mitchellh/cli"
	"os"
)

func Commands() map[string]cli.CommandFactory {
	ui := &cli.BasicUi{
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}
	return map[string]cli.CommandFactory{
		"config": func() (cli.Command, error) {
			return &ConfigCommand{
				ui: ui,
			}, nil
		},
	}
}
