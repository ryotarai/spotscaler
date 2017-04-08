package cli

import (
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

	ec2, err := spotscaler.NewEC2(config.WorkingFilters, config.SpotProductDescription, config.SpotVarieties())
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
	for az, a := range config.SpotLaunchMethods {
		for t, m := range a {
			v := spotscaler.InstanceVariety{
				InstanceType:     t,
				AvailabilityZone: az,
			}
			if state.SpotPrice[v] <= m.BiddingPrice {
				capacityByVariety[v] = m.Capacity
			}
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

	simulator, err := spotscaler.NewSimulator(metric, config.Threshold, capacityByVariety, config.PossibleTermination, config.InitialCapacity, config.ScalingInFactor, 0, config.MaximumCapacity)
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
