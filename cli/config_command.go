package cli

import (
	"flag"
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/config"
)

type ConfigCommand struct {
	ui cli.Ui
}

func (c *ConfigCommand) Help() string {
	return ""
}

func (c *ConfigCommand) Run(args []string) int {
	flags := flag.NewFlagSet("spotscaler", flag.ContinueOnError)
	path := flags.String("-config-path", "", "path")
	if err := flags.Parse(args); err != nil {
		return 1
	}

	if *path == "" {
		c.ui.Error("-config-path is required")
		return 1
	}

	cc, err := config.LoadFromYAMLPath(*path)
	if err != nil {
		c.ui.Error(fmt.Sprint(err))
		return 1
	}

	err = config.Validate(cc)
	if err != nil {
		c.ui.Error("Validation error:")
		c.ui.Error(fmt.Sprint(err))
		return 1
	}

	c.ui.Output(fmt.Sprintf("%+v", cc))

	return 0
}

func (c *ConfigCommand) Synopsis() string {
	return "Show config in parsed format"
}
