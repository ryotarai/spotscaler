package config

type Config struct {
	ID                     string              `yaml:"ID" validate:"required"`
	LaunchConfiguration    LaunchConfiguration `yaml:"LaunchConfiguration" validate:"required"`
	WorkingInstanceFilters []EC2Filter         `yaml:"WorkingInstanceFilters" validate:"dive"`
	TerminateTags          map[string]string   `yaml:"TerminateTags" validate:"required"`
	InstanceTags           map[string]string   `yaml:"InstanceTags"`
	InstanceCapacityByType map[string]int      `yaml:"InstanceCapacityByType" validate:"required"`
	BiddingPriceByType     map[string]float64  `yaml:"BiddingPriceByType" validate:"required"`
	RedisHost              string              `yaml:"RedisHost" validate:"required"`
	CooldownDuration       string              `yaml:"Cooldown" validate:"required"`
	HookCommands           []Command           `yaml:"HookCommands"`
	AMICommand             Command             `yaml:"AMICommand" validate:"required"`
	CPUUtilCommand         Command             `yaml:"CPUUtilCommand" validate:"required"`
	CapacityTagKey         string              `yaml:"CapacityTagKey"`
	Timers                 map[string]Timer    `yaml:"Timers" validate:"dive"`
	MaxCPUUtil             float64             `yaml:"MaxCPUUtil" validate:"required"`
	MaxCapacity            float64             `yaml:"MaxCapacity"`
	MaxTerminatedVarieties int                 `yaml:"MaxTerminatedVarieties" validate:"required"`
	ScaleInThreshold       float64             `yaml:"ScaleInThreshold" validate:"required"`
	HTTPAddr               string              `yaml:"HTTPAddr"`
}

func NewConfig() *Config {
	return &Config{}
}
