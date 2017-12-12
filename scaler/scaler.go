package scaler

import (
	"net/http"
	"time"

	"github.com/ryotarai/spotscaler/httpapi"
	"github.com/sirupsen/logrus"
)

type Scaler struct {
	logger *logrus.Logger
	config *Config
	api    *httpapi.Handler
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
		api:    httpapi.NewHandler(),
	}, nil
}

func (s *Scaler) Start() {
	s.logger.Infof("Starting Spotscaler v%s", Version)
	s.logger.Debugf("Loaded config is %#v", s.config)

	if s.config.APIAddr != "" {
		s.StartAPIServer()
	}

	for {
		err := s.Run()
		if err != nil {
			s.logger.Error(err)
		}

		s.logger.Info("Waiting for next run")
		time.Sleep(60 * time.Second)
	}
}

func (s *Scaler) StartAPIServer() {
	sv := &http.Server{
		Addr:    s.config.APIAddr,
		Handler: s.api,
	}
	s.logger.Infof("Starting HTTP API on %s", s.config.APIAddr)

	go func() {
		err := sv.ListenAndServe()
		if err != nil {
			s.logger.Error(err)
		}
	}()
}

func (s *Scaler) Run() error {
	metric, err := s.config.MetricCommand.GetFloat()
	if err != nil {
		return err
	}
	s.logger.Debugf("Metric value: %f", metric)
	s.api.UpdateMetric("metric", metric)

	return nil
}
