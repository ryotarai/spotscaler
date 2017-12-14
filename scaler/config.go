package scaler

import (
	"io/ioutil"

	"github.com/ryotarai/spotscaler/command"
	"github.com/ryotarai/spotscaler/ec2"
	"github.com/ryotarai/spotscaler/timer"
	validator "gopkg.in/go-playground/validator.v9"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel               string                    `yaml:"LogLevel"`
	DryRun                 bool                      `yaml:"DryRun"`
	AMICommand             *command.Command          `yaml:"AMICommand" validate:"required"`
	MetricCommand          *command.Command          `yaml:"MetricCommand" validate:"required"`
	EventCommand           *command.Command          `yaml:"EventCommand"`
	Timers                 []*timer.Timer            `yaml:"Timers"`
	APIAddr                string                    `yaml:"APIAddr"`
	CapacityTagKey         string                    `yaml:"CapacityTagKey" validate:"required"`
	WorkingFilters         map[string][]string       `yaml:"WorkingFilters" validate:"required"`
	PossibleTermination    int64                     `yaml:"PossibleTermination" validate:"required"`
	InstanceTypes          []string                  `yaml:"InstanceTypes" validate:"required"`
	AvailabilityZones      []string                  `yaml:"AvailabilityZones" validate:"required"`
	CapacityByType         map[string]float64        `yaml:"CapacityByType" validate:"required"`
	ScalingOutThreshold    float64                   `yaml:"ScalingOutThreshold" validate:"required"`
	ScalingInThreshold     float64                   `yaml:"ScalingInThreshold" validate:"required"`
	MinimumCapacity        float64                   `yaml:"MinimumCapacity" validate:"required"`
	SubnetByAZ             map[string]string         `yaml:"SubnetByAZ" validate:"required"`
	KeyName                string                    `yaml:"KeyName" validate:"required"`
	SecurityGroupIDs       []string                  `yaml:"SecurityGroupIDs" validate:"required"`
	UserData               string                    `yaml:"UserData"`
	IAMInstanceProfileName string                    `yaml:"IAMInstanceProfileName"`
	BlockDeviceMappings    []*ec2.BlockDeviceMapping `yaml:"BlockDeviceMappings"`
	RedisURL               string                    `yaml:"RedisURL" validate:"required"`
	RedisKeyPrefix         string                    `yaml:"RedisKeyPrefix"`
}

func NewConfig() *Config {
	return &Config{
		LogLevel: "info",
	}
}

func NewConfigFromFile(path string) (*Config, error) {
	var err error

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	c := NewConfig()
	err = yaml.UnmarshalStrict(b, c)
	if err != nil {
		return nil, err
	}

	err = c.Validate()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) Validate() error {
	validate := validator.New()
	err := validate.Struct(c)
	if err != nil {
		return err
	}
	return nil
}
