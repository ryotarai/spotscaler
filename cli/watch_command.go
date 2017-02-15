package cli

import (
	"flag"
	"fmt"

	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/api"
	"github.com/ryotarai/spotscaler/config"
	"github.com/ryotarai/spotscaler/ec2"
	"github.com/ryotarai/spotscaler/scaler"
	"github.com/ryotarai/spotscaler/state"
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

	state := state.NewRedisState(cc.RedisHost)

	apiServer := &api.Server{
		Ui:    c.ui,
		State: state,
		Addr:  cc.HTTPAddr,
	}
	apiServer.Start() // background

	ec2, err := ec2.NewClient(c.ui)
	if err != nil {
		c.ui.Error(fmt.Sprint(err))
		return 1
	}

	scaler := &scaler.Scaler{
		EC2:    ec2,
		Ui:     c.ui,
		State:  state,
		Config: cc,
	}
	err = scaler.Start()
	if err != nil {
		c.ui.Error(fmt.Sprint(err))
		return 1
	}

	return 0
}

func (c *WatchCommand) Synopsis() string {
	return "Watch instance state and scale automatically"
}
