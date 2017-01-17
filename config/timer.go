package config

type Timer struct {
	Command  `yaml:"Command" validate:"required,dive"`
	After    string `yaml:"After" validate:"required"`
	Duration string `yaml:"Duration" validate:"required"`
}
