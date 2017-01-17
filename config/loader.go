package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

func LoadFromYAMLPath(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return LoadFromYAML(data)
}

func LoadFromYAML(data []byte) (*Config, error) {
	config := NewConfig()
	err := yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
