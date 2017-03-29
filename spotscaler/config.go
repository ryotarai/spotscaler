package spotscaler

import (
	"io/ioutil"

	validator "gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

type Config struct {
	WorkingFilters map[string][]string `yaml:"workingFilters" validate:"required"`
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
