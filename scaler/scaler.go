package scaler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ryotarai/spotscaler/ec2"
	"github.com/ryotarai/spotscaler/httpapi"
	"github.com/ryotarai/spotscaler/simulator"
	"github.com/ryotarai/spotscaler/storage"
	"github.com/ryotarai/spotscaler/timer"
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
	s.runTimers()

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
		if desiredInstances == nil || d.TotalCapacity() > desiredInstances.TotalCapacity() {
			desiredInstances = d
		}
	}

	schedule, err := s.storage.ActiveSchedule()
	if err != nil {
		return err
	}

	if schedule != nil {
		s.logger.Infof("An active schedule is found: %#v", schedule)
		d := s.simulator.DesiredInstancesFromCapacity(instances, schedule.Capacity)
		if instances.TotalCapacity() < d.TotalCapacity() && (desiredInstances == nil || d.TotalCapacity() > desiredInstances.TotalCapacity()) {
			desiredInstances = d
		}
	}

	if desiredInstances != nil {
		err := s.fireScalingEvent(instances, desiredInstances)
		if err != nil {
			return err
		}

		if desiredInstances.TotalCapacity() < instances.TotalCapacity() {
			s.logger.Info("Start to scale in")
			s.ec2.TerminateInstances(instances, desiredInstances)
		} else if desiredInstances.TotalCapacity() > instances.TotalCapacity() {
			s.logger.Info("Start to scale out")

			s.logger.Debug("Retrieving AMI")
			ami, err := s.config.AMICommand.GetString("")
			if err != nil {
				return err
			}
			if ami == "" {
				s.logger.Warn("AMI is not found")
				return nil
			}

			s.ec2.LaunchInstances(desiredInstances, ami)
		}

		s.setTimers("LaunchingInstances")
	}

	return nil
}

func (s *Scaler) updateMetric(k string, v float64) {
	s.logger.Debugf("Updating a metric %s:%f", k, v)
	s.api.UpdateMetric(k, v)
}

func (s *Scaler) fireScalingEvent(from ec2.Instances, to ec2.Instances) error {
	if s.config.EventCommand == nil {
		return nil
	}

	type event struct {
		Event string
		From  ec2.Instances
		To    ec2.Instances
	}
	e := event{"scaling", from, to}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = s.config.EventCommand.GetString(fmt.Sprintf("%s\n", string(b)))
	return err
}

func (s *Scaler) setTimers(after string) error {
	for _, t := range s.config.Timers {
		if t.After == after {
			err := s.storage.RegisterTimer(t)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Scaler) runTimers() error {
	names, err := s.storage.GetExpiredTimerNames()
	if err != nil {
		return err
	}

	for _, n := range names {
		var timer *timer.Timer
		for _, t := range s.config.Timers {
			if t.Name == n {
				timer = t
			}
		}

		if timer == nil {
			s.logger.Warnf("Timer '%s' is not found", n)
			continue
		}

		s.logger.Infof("Running a timer: %#v", timer)
		_, err := timer.Command.GetString("")
		if err != nil {
			return err
		}

		err = s.storage.DeregisterTimer(timer)
		if err != nil {
			return err
		}
	}

	return nil
}
