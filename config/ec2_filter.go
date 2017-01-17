package config

type EC2Filter struct {
	Name   string   `yaml:"Name" validate:"required"`
	Values []string `yaml:"Values" validate:"required"`
}
