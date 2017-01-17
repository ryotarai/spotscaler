package config

type Command struct {
	Command string   `yaml:"Command" validate:"required"`
	Args    []string `yaml:"Args"`
}
