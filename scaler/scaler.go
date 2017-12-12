package scaler

import (
	"time"

	"github.com/sirupsen/logrus"
)

type Scaler struct {
	logger *logrus.Logger
	config *Config
}

func NewScaler(c *Config) (*Scaler, error) {
	lv, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		return nil, err
	}

	logger := logrus.New()
	logger.Level = lv

	return &Scaler{
		logger: logger,
		config: c,
	}, nil
}

func (s *Scaler) Start() {
	s.logger.Infof("Starting Spotscaler v%s", Version)
	s.logger.Debugf("Loaded config is %#v", s.config)

	for {
		err := s.Run()
		if err != nil {
			s.logger.Error(err)
		}

		s.logger.Info("Waiting for next run")
		time.Sleep(60 * time.Second)
	}
}

func (s *Scaler) Run() error {
	metric, err := s.config.MetricCommand.GetFloat()
	if err != nil {
		return err
	}
	s.logger.Debugf("Metric value: %f", metric)

	return nil
}
