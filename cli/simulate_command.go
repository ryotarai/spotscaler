package cli

import (
	"log"

	"flag"

	"fmt"

	"github.com/ryotarai/spotscaler/spotscaler"
)

type simulateCommand struct {
	logger *log.Logger
}

func (c *simulateCommand) Help() string {
	return "Show simulation"
}

func (c *simulateCommand) Synopsis() string {
	return "Show simulation"
}

func (c *simulateCommand) Run(args []string) int {
	configPath, verbose, err := parseFlags(args)
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	config, err := spotscaler.LoadConfigYAML(*configPath)
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	if *verbose {
		c.logger.Println(config)
	}

	c.logger.Println("Starting simulation")

	ec2, err := spotscaler.NewEC2()
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	state, err := ec2.GetCurrentState(config.WorkingFilters)
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	c.logger.Println("# Running instances")
	for _, i := range state.Instances {
		c.logger.Printf("- %s", i.InstanceID)
	}

	// state := spotscaler.GetCurrentState()

	// simulator := spotscaler.NewSimulator()
	// result := simulator.Simulate(state)

	return 0
}

func parseFlags(args []string) (*string, *bool, error) {
	fs := flag.NewFlagSet("spotscaler", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to config YAML file")
	verbose := fs.Bool("verbose", false, "Output detailed log")
	err := fs.Parse(args)
	if err != nil {
		return nil, nil, err
	}

	if *configPath == "" {
		return nil, nil, fmt.Errorf("-config option is mandatory")
	}

	return configPath, verbose, nil
}
