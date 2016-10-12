package autoscaler

type Timer struct {
	Command  `yaml:"Command"`
	After    string `yaml:"After" validate:"required"`
	Duration string `yaml:"Duration" validate:"required"`
}
