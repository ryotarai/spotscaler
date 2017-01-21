package config

type InstanceType struct {
	InstanceTypeName string  `yaml:"InstanceTypeName" validate:"required"`
	Capacity         int     `yaml:"Capacity" validate:"required"`
	BiddingPrice     float64 `yaml:"BiddingPrice" validate:"required"`
}
