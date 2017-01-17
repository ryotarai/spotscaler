package config

import (
	"gopkg.in/go-playground/validator.v9"
)

func Validate(c *Config) error {
	v := validator.New()
	return v.Struct(c)
}
