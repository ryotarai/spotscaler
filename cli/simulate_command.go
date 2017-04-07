package cli

import (
	"flag"
	"fmt"
	"log"
	"strconv"

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
	configPath, err := parseFlags(args)
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	config, err := spotscaler.LoadConfigYAML(*configPath)
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	c.logger.Println("starting simulation")

	varieties := []spotscaler.InstanceVariety{}
	for v := range config.CapacityByVariety() {
		varieties = append(varieties, v)
	}

	ec2, err := spotscaler.NewEC2(config.WorkingFilters, config.SpotProductDescription, varieties)
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	state, err := ec2.GetCurrentState()
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	c.logger.Printf("spot price: %v", state.SpotPrice)
	capacityByVariety := map[spotscaler.InstanceVariety]int{}
	for v, cap := range config.CapacityByVariety() {
		bid := config.SpotBiddingPrice[v.InstanceType]
		if bid == 0.0 {
			c.logger.Printf("bidding price for %s is not configured", v.InstanceType)
			return 1
		}

		if state.SpotPrice[v] <= bid {
			capacityByVariety[v] = cap
		}
	}

	s, err := config.MetricCommand.Run()
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	metric, err := strconv.ParseFloat(s, 64)
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	c.logger.Printf("current metric: %f", metric)

	simulator, err := spotscaler.NewSimulator(metric, config.Threshold, config.CapacityByVariety(), config.PossibleTermination, config.InitialCapacity, config.ScalingInFactor, 0, config.MaximumCapacity)
	if err != nil {
		c.logger.Println(err)
		return 1
	}

	terminate, keep, launch, err := simulator.Simulate(state)

	c.logger.Println("-: will be terminated, *: will be kept, +: will be launched")
	for _, i := range terminate {
		c.logger.Printf("-: %+v (%d, %s)", i.Variety, i.Capacity, i.InstanceID)
	}
	for _, i := range keep {
		c.logger.Printf("*: %+v (%d, %s)", i.Variety, i.Capacity, i.InstanceID)
	}
	for _, i := range launch {
		c.logger.Printf("+: %+v (%d)", i.Variety, i.Capacity)
	}

	return 0
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
