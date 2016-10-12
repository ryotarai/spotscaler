package autoscaler

import (
	"time"
)

type Schedule struct {
	Key      string
	StartAt  time.Time `binding:"required"`
	EndAt    time.Time `binding:"required"`
	Capacity float64   `binding:"required"`
}

func NewSchedule() *Schedule {
	return &Schedule{
		Key: time.Now().UTC().Format(time.RFC3339Nano),
	}
}
