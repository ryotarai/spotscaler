package spotscaler

import (
	"io/ioutil"

	validator "gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Threshold              float64                                `yaml:"threshold" validate:"required"`
	WorkingFilters         map[string][]string                    `yaml:"workingFilters" validate:"required"`
	PossibleTermination    int                                    `yaml:"possibleTermination" validate:"required"`
	MetricCommand          *Command                               `yaml:"metricCommand" validate:"required,dive"`
	InitialCapacity        int                                    `yaml:"initialCapacity" validate:"required"`
	MaximumCapacity        int                                    `yaml:"maximumCapacity" validate:"required"`
	ScalingInFactor        float64                                `yaml:"scalingInFactor" validate:"required"`
	SpotProductDescription string                                 `yaml:"spotProductDescription" validate:"required"`
	SpotLaunchMethods      map[string]map[string]SpotLaunchMethod `yaml:"spotLaunchMethods" validate:"required"`
}

type SpotLaunchMethod struct {
	BiddingPrice float64 `yaml:"biddingPrice" validate:"required"`
	Capacity     int     `yaml:"capacity" validate:"required"`
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

func (c *Config) SpotVarieties() []InstanceVariety {
	ret := []InstanceVariety{}
	for az, a := range c.SpotLaunchMethods {
		for t := range a {
			v := InstanceVariety{
				AvailabilityZone: az,
				InstanceType:     t,
			}
			ret = append(ret, v)
		}
	}
	return ret
}
