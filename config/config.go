package config

type Config struct {
	ID                       string              `yaml:"ID" validate:"required"`
	LaunchConfiguration      LaunchConfiguration `yaml:"LaunchConfiguration" validate:"required"`
	WorkingInstanceFilters   []EC2Filter         `yaml:"WorkingInstanceFilters" validate:"dive"`
	TerminateTags            map[string]string   `yaml:"TerminateTags" validate:"required"`
	InstanceTags             map[string]string   `yaml:"InstanceTags"`
	RedisHost                string              `yaml:"RedisHost" validate:"required"`
	CooldownDuration         string              `yaml:"Cooldown" validate:"required"`
	HookCommands             []Command           `yaml:"HookCommands"`
	AMICommand               Command             `yaml:"AMICommand" validate:"required"`
	Timers                   map[string]Timer    `yaml:"Timers" validate:"dive"`
	MetricCommand            Command             `yaml:"MetricCommand" validate:"required"`
	ScalingOutThreshold      float64             `yaml:"ScalingOutThreshold" validate:"required"`
	ScalingInThresholdFactor float64             `yaml:"ScalingInThresholdFactor" validate:"required"`
	MaxTotalCapacity         int                 `yaml:"MaxTotalCapacity"`
	MaxTerminatedVarieties   int                 `yaml:"MaxTerminatedVarieties" validate:"required"`
	HTTPAddr                 string              `yaml:"HTTPAddr"`
}

func NewConfig() *Config {
	return &Config{}
}
