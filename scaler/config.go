package scaler

import (
	"io/ioutil"

	"github.com/ryotarai/spotscaler/command"
	validator "gopkg.in/go-playground/validator.v9"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel      string           `yaml:"LogLevel"`
	MetricCommand *command.Command `yaml:"MetricCommand" validate:"required"`
	APIAddr       string           `yaml:"APIAddr"`
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
