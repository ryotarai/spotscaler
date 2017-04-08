package cli

import (
	"flag"
	"fmt"
	"log"

	"os"

	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/spotscaler"
)

// Start CLI
func Start(args []string) int {
	c := cli.NewCLI("spotscaler", spotscaler.Version)
	c.Args = args[1:]
	c.Commands = commands()

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	return exitStatus
}

func commands() map[string]cli.CommandFactory {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	return map[string]cli.CommandFactory{
		"version": func() (cli.Command, error) {
			return &versionCommand{}, nil
		},
		"simulate": func() (cli.Command, error) {
			return &simulateCommand{
				logger: logger,
			}, nil
		},
	}
}

func parseFlags(args []string) (*string, error) {
	fs := flag.NewFlagSet("spotscaler", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to config YAML file")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	if *configPath == "" {
		return nil, fmt.Errorf("-config option is mandatory")
	}

	return configPath, nil
}
