package spotscaler

import (
	"io/ioutil"

	validator "gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Threshold              float64             `yaml:"threshold" validate:"required"`
	WorkingFilters         map[string][]string `yaml:"workingFilters" validate:"required"`
	CapacityByType         map[string]int      `yaml:"capacityByType" validate:"required"`
	AvailabilityZones      []string            `yaml:"availabilityZones" validate:"required"`
	PossibleTermination    int                 `yaml:"possibleTermination" validate:"required"`
	MetricCommand          *Command            `yaml:"metricCommand" validate:"required,dive"`
	InitialCapacity        int                 `yaml:"initialCapacity" validate:"required"`
	MaximumCapacity        int                 `yaml:"maximumCapacity" validate:"required"`
	ScalingInFactor        float64             `yaml:"scalingInFactor" validate:"required"`
	SpotProductDescription string              `yaml:"spotProductDescription" validate:"required"`
	SpotBiddingPrice       map[string]float64  `yaml:"spotBiddingPrice" validate:"required"`
}

func LoadConfigYAML(path string) (*Config, error) {
	in, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	err = yaml.Unmarshal(in, config)
	if err != nil {
		return nil, err
	}

	v := validator.New()
	err = v.Struct(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) CapacityByVariety() map[InstanceVariety]int {
	ret := map[InstanceVariety]int{}
	for _, az := range c.AvailabilityZones {
		for t, c := range c.CapacityByType {
			v := InstanceVariety{
				AvailabilityZone: az,
				InstanceType:     t,
			}
			ret[v] = c
		}
	}
	return ret
}
