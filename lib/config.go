package autoscaler

import (
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

// Config represents configuration loaded from a file
type Config struct {
	LaunchConfiguration    LaunchConfiguration `yaml:"LaunchConfiguration" validate:"required"`
	WorkingInstanceFilters EC2Filters          `yaml:"WorkingInstanceFilters" validate:"dive"`
	TerminateTags          EC2Tags             `yaml:"TerminateTags" validate:"required,dive"`
	InstanceTags           EC2Tags             `yaml:"InstanceTags" validate:"dive"`
	LoopInterval           string              `yaml:"LoopInterval" validate:"required"`
	InstanceCapacityByType map[string]float64  `yaml:"InstanceCapacityByType" validate:"required"`
	BiddingPriceByType     map[string]float64  `yaml:"BiddingPriceByType" validate:"required"`
	InstanceTypes          []string            `yaml:"InstanceTypes" validate:"required"`
	Subnets                []Subnet            `yaml:"Subnets" validate:"required,dive"`
	RedisHost              string              `yaml:"RedisHost" validate:"required"`
	RedisKeyPrefix         string              `yaml:"RedisKeyPrefix"`
	Cooldown               string              `yaml:"Cooldown" validate:"required"`
	HookCommands           []Command           `yaml:"HookCommands"`
	AMICommand             Command             `yaml:"AMICommand" validate:"required"`
	CPUUtilCommand         Command             `yaml:"CPUUtilCommand" validate:"required"`
	CapacityTagKey         string              `yaml:"CapacityTagKey"`
	ConfirmBeforeAction    bool                `yaml:"ConfirmBeforeAction"`
	Timers                 map[string]Timer    `yaml:"Timers" validate:"dive"`
	MaximumCPUUtil         float64             `yaml:"MaximumCPUUtil" validate:"required"`
	MaximumCapacity        float64             `yaml:"MaximumCapacity"`
	AcceptableTermination  int                 `yaml:"AcceptableTermination" validate:"required"`
	ScaleInThreshold       float64             `yaml:"ScaleInThreshold" validate:"required"`
	DryRun                 bool                `yaml:"DryRun"`
}

func (c *Config) InstanceVarieties() []InstanceVariety {
	vs := []InstanceVariety{}
	for _, t := range c.InstanceTypes {
		for _, s := range c.Subnets {
			v := InstanceVariety{
				Subnet:       s,
				InstanceType: t,
			}
			vs = append(vs, v)
		}
	}

	return vs
}

// Validate validates config data
func (c *Config) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// LoadYAMLConfig loads from YAML file and returns Config
func LoadYAMLConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	config := Config{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
