package cli

import (
	"flag"
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/config"
)

type WatchCommand struct {
	ui cli.Ui
}

func (c *WatchCommand) Help() string {
	return ""
}

func (c *WatchCommand) Run(args []string) int {
	flags := flag.NewFlagSet("spotscaler", flag.ContinueOnError)
	configPath := flags.String("config-path", "", "config path")
	if err := flags.Parse(args); err != nil {
		return 1
	}

	if *configPath == "" {
		c.ui.Error("-config-path is required")
		return 1
	}

	cc, err := config.LoadFromYAMLPath(*configPath)
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

	c.ui.Info(fmt.Sprintf("Loaded config: %+v", cc))

	// state := state.NewState(cc.RedisHost)
	// apiServer := api.NewServer(state)
	// apiServer.Start()
	// watcher := watcher.NewWatcher(state, config)
	// watcher.Start()

	return 0
}

func (c *WatchCommand) Synopsis() string {
	return "Watch instance state and scale automatically"
}
