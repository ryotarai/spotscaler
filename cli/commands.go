package cli

import (
	"github.com/mitchellh/cli"
	"os"
)

func Commands() map[string]cli.CommandFactory {
	ui := &cli.PrefixedUi{
		Ui: &cli.BasicUi{
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
		OutputPrefix: "DEBUG: ",
		InfoPrefix:   "INFO:  ",
		ErrorPrefix:  "ERROR: ",
		WarnPrefix:   "WARN:  ",
	}

	return map[string]cli.CommandFactory{
		"config": func() (cli.Command, error) {
			return &ConfigCommand{
				ui: ui,
			}, nil
		},
		"watch": func() (cli.Command, error) {
			return &WatchCommand{
				ui: ui,
			}, nil
		},
	}
}
