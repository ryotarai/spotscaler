package autoscaler

import (
	"fmt"
	"log"
	"sort"
)

type ScalingPolicy struct {
	If         string  `yaml:"If" validate:"required"`
	Threshold  float64 `yaml:"Threshold" validate:"required"`
	MetricType string  `yaml:"MetricType" validate:"required"`
	Target     float64 `yaml:"Target" validate:"required"`
}

func (p ScalingPolicy) Rate(values []float64) (float64, error) {
	sort.Float64s(values)

	var metric float64
	switch p.MetricType {
	case "median":
		metric = values[len(values)/2]
	case "max":
		metric = values[len(values)-1]
	default:
		return 1.0, fmt.Errorf("Metric type %s is invalid", p.MetricType)
	}

	var match bool
	switch p.If {
	case "greaterThan":
		match = p.Threshold < metric
	case "lessThan":
		match = p.Threshold > metric
	default:
		return 1.0, fmt.Errorf("%s is invalid If value", p.If)
	}

	if match {
		log.Println("[DEBUG] matched policy:", p)
		log.Printf("[DEBUG] %f -> %f", metric, p.Target)
		return metric / p.Target, nil
	}

	return 1.0, nil // not matched
}
