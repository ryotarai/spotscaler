package scaler

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel string `yaml:"LogLevel"`
}

func NewConfigFromFile(path string) (*Config, error) {
	var err error

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.UnmarshalStrict(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
