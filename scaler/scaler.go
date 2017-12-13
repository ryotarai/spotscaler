package scaler

import (
	"net/http"
	"time"

	"github.com/ryotarai/spotscaler/ec2"
	"github.com/ryotarai/spotscaler/httpapi"
	"github.com/ryotarai/spotscaler/simulator"
	"github.com/ryotarai/spotscaler/storage"
	"github.com/sirupsen/logrus"
)

type Scaler struct {
	logger    *logrus.Logger
	config    *Config
	api       *httpapi.Handler
	ec2       *ec2.EC2
	simulator *simulator.Simulator
	storage   storage.Storage
}

func NewScaler(c *Config) (*Scaler, error) {
	lv, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		return nil, err
	}

	logger := logrus.New()
	logger.Level = lv

	e := ec2.New()
	e.CapacityTagKey = c.CapacityTagKey
	e.WorkingFilters = c.WorkingFilters
	e.Logger = logger
	e.SubnetByAZ = c.SubnetByAZ
	e.KeyName = c.KeyName
	e.SecurityGroupIDs = c.SecurityGroupIDs
	e.UserData = c.UserData
	e.IAMInstanceProfileName = c.IAMInstanceProfileName
	e.BlockDeviceMappings = c.BlockDeviceMappings
	e.DryRun = c.DryRun

	storage, err := storage.NewRedisStorage(c.RedisURL, c.RedisKeyPrefix)
	if err != nil {
		return nil, err
	}

	return &Scaler{
		logger:  logger,
		config:  c,
		api:     httpapi.NewHandler(storage),
		ec2:     e,
		storage: storage,
		simulator: &simulator.Simulator{
			Logger:              logger,
			PossibleTermination: c.PossibleTermination,
			InstanceTypes:       c.InstanceTypes,
			AvailabilityZones:   c.AvailabilityZones,
			CapacityByType:      c.CapacityByType,
		},
	}, nil
}

func (s *Scaler) Start() {
	s.logger.Infof("Starting Spotscaler v%s", Version)
	s.logger.Debugf("Loaded config is %#v", s.config)

	if s.config.APIAddr != "" {
		s.StartAPIServer()
	}

	s.updateMetric("scaling_out_threshold", s.config.ScalingOutThreshold)
	s.updateMetric("scaling_in_threshold", s.config.ScalingInThreshold)

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
	s.updateMetric("metric", metric)

	s.logger.Debug("Getting working instances")
	instances, err := s.ec2.GetInstances()
	if err != nil {
		return err
	}
	s.updateMetric("total_capacity", instances.TotalCapacity())

	worstInstances := s.simulator.WorstCase(instances)
	s.updateMetric("worst_total_capacity", worstInstances.TotalCapacity())

	worstMetric := metric * worstInstances.TotalCapacity() / instances.TotalCapacity()
	s.updateMetric("worst_metric", worstMetric)

	var desiredInstances ec2.Instances

	minInstances := s.simulator.DesiredInstancesFromCapacity(instances, s.config.MinimumCapacity)
	if instances.TotalCapacity() < minInstances.TotalCapacity() {
		s.logger.Info("Current capacity is less than the minimum capacity")
		desiredInstances = minInstances
	}

	if instances.TotalCapacity() > 0 && (worstMetric < s.config.ScalingInThreshold || s.config.ScalingOutThreshold < worstMetric) {
		d := s.simulator.DesiredInstancesFromMetric(instances, worstMetric)
		if d.TotalCapacity() > desiredInstances.TotalCapacity() {
			desiredInstances = d
		}
	}

	if desiredInstances != nil {
		if desiredInstances.TotalCapacity() < instances.TotalCapacity() {
			s.logger.Info("Trying to scaling in")
			s.ec2.TerminateInstances(instances, desiredInstances)
		} else if desiredInstances.TotalCapacity() > instances.TotalCapacity() {
			s.logger.Info("Trying to scaling out")
			s.ec2.LaunchInstances(desiredInstances, "dummy")
		}
	}

	return nil
}

func (s *Scaler) updateMetric(k string, v float64) {
	s.logger.Debugf("Updating a metric %s:%f", k, v)
	s.api.UpdateMetric(k, v)
}
