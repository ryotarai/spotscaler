package autoscaler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"os"
	"strings"
	"time"
)

type Runner struct {
	config         *Config
	status         StatusStoreIface
	awsSession     *session.Session
	ec2Client      EC2ClientIface
	metricProvider MetricProvider
}

func NewRunner(config *Config) (*Runner, error) {
	awsSess, err := session.NewSession(aws.NewConfig())
	if err != nil {
		return nil, err
	}

	metric, err := NewMetricProvider("CloudWatchEC2", awsSess)
	if err != nil {
		return nil, err
	}

	runner := &Runner{
		config:         config,
		status:         NewStatusStore(config.RedisHost, config.RedisKeyPrefix),
		awsSession:     awsSess,
		ec2Client:      NewEC2Client(ec2.New(awsSess), config),
		metricProvider: metric,
	}

	return runner, nil
}

func (r *Runner) StartLoop() error {
	SetCapacityTable(r.config.InstanceCapacityByType)

	loopInterval, err := time.ParseDuration(r.config.LoopInterval)
	if err != nil {
		return err
	}

	for {
		c := time.After(loopInterval)

		if err != nil {
			log.Println("[ERROR] error in loop:", err)
		} else {
			err := r.Run()
			if err != nil {
				log.Println("[ERROR] error in loop:", err)
			}
		}

		log.Println("[INFO] waiting for next run")
		<-c
	}

	return nil
}

func (r *Runner) Run() error {
	var err error

	log.Println("[DEBUG] START Runner.Run")
	if err != nil {
		return err
	}

	err = r.runExpiredTimers()
	if err != nil {
		return err
	}

	err = r.propagateSIRTagsToInstances()
	if err != nil {
		return err
	}

	cooldownEndsAt, err := r.status.FetchCooldownEndsAt()
	if err != nil {
		return err
	}

	if time.Now().Before(cooldownEndsAt) {
		log.Printf("[INFO] in cooldown (it ends at %s)", cooldownEndsAt)
		return nil
	}

	err = r.scale()
	if err != nil {
		return err
	}

	log.Println("[DEBUG] END Runner.Run")
	return nil
}

func (r *Runner) propagateSIRTagsToInstances() error {
	log.Println("[DEBUG] START: propagateSIRTagsToInstances")
	// find active and status:pending SIRs
	pendingSIRs, err := r.ec2Client.DescribePendingAndActiveSIRs()
	if err != nil {
		return err
	}

	if len(pendingSIRs) == 0 {
		log.Println("[INFO] no active and pending spot instance requests")
		return nil
	}

	log.Println("[INFO] propagating tags from spot instance requests")

	// propagate tags
	err = r.ec2Client.PropagateTagsFromSIRsToInstances(pendingSIRs)
	if err != nil {
		return err
	}

	// status:completed tag to SIR
	err = r.ec2Client.CreateStatusTagsOfSIRs(pendingSIRs, "completed")
	if err != nil {
		return err
	}

	return nil
}

func (r *Runner) scale() error {
	log.Println("[DEBUG] START: scale")

	workingInstances, err := r.ec2Client.DescribeWorkingInstances()
	if err != nil {
		return err
	}

	ondemandCapacity, err := workingInstances.Ondemand().Capacity()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] ondemand capacity: %f", ondemandCapacity.Total())

	spotCapacity, err := workingInstances.Spot().Capacity()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] spot capacity: %f", ondemandCapacity.Total())

	price, err := r.ec2Client.DescribeSpotPrices(r.config.InstanceVarieties)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] current spot price: %v", price)

	availableVarieties := []InstanceVariety{}
	for v, p := range price {
		bid, ok := r.config.BiddingPriceByType[v.InstanceType]
		if !ok {
			return fmt.Errorf("Bidding price for %s is unknown", v.InstanceType)
		}

		if p <= bid {
			availableVarieties = append(availableVarieties, v)
		} else {
			log.Printf("[DEBUG] %v is not available due to price (%f USD)", v, p)
		}
	}

	schedule, err := r.getCurrentSchedule()
	if err != nil {
		return err
	}

	var totalDesiredCapacity float64
	if schedule == nil {
		keepRateOfSpot := float64(len(availableVarieties)-r.config.AcceptableTermination) / float64(len(availableVarieties))
		cpuUtilToScaleOut := r.config.MaximumCPUUtil *
			(ondemandCapacity.Total() + spotCapacity.Total()*keepRateOfSpot) /
			(ondemandCapacity.Total() + spotCapacity.Total())
		cpuUtilToScaleIn := cpuUtilToScaleOut * r.config.RateOfCPUUtilToScaleIn
		log.Printf("[DEBUG] cpu util to scale out: %f, cpu util to scale in: %f", cpuUtilToScaleOut, cpuUtilToScaleIn)

		metric, err := r.metricProvider.Values(workingInstances)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] max of metric: %f, median of metric: %f", metric.Max(), metric.Median())

		var cpuUtil float64
		if metric.Max() <= cpuUtilToScaleIn {
			log.Println("[DEBUG] scaling in")
			cpuUtil = metric.Max()
		} else if cpuUtilToScaleOut <= metric.Median() {
			log.Println("[DEBUG] scaling out")
			cpuUtil = metric.Median()
		} else {
			log.Println("[DEBUG] skip both scaling in and scaling out")
			return nil
		}

		scalingRate := ((((2*cpuUtil*(ondemandCapacity.Total()+spotCapacity.Total()))/(r.config.MaximumCPUUtil*(1+r.config.RateOfCPUUtilToScaleIn)) - ondemandCapacity.Total()) / keepRateOfSpot) + ondemandCapacity.Total()) / (ondemandCapacity.Total() + spotCapacity.Total())
		scalingRate = r.correctScalingRate(scalingRate)
		log.Printf("[INFO] scaling rate: %f", scalingRate)
		log.Printf("[INFO] expected CPU util after scaling: %f", cpuUtil/scalingRate)

		totalDesiredCapacity = (ondemandCapacity.Total() + spotCapacity.Total()) * scalingRate
		totalDesiredCapacity = r.correctDesiredTotalCapacity(totalDesiredCapacity)
	} else {
		log.Println("[INFO] schedule found:", schedule)
		totalDesiredCapacity = schedule.Capacity
	}
	log.Printf("[DEBUG] total desired capacity: %f", totalDesiredCapacity)

	totalDesiredSpotCapacity := totalDesiredCapacity - ondemandCapacity.Total()
	if totalDesiredSpotCapacity < 0.0 {
		log.Printf("[DEBUG] total desired spot capacity (%f) is less than 0, fixing it to 0 forcibly", totalDesiredSpotCapacity)
		totalDesiredSpotCapacity = 0.0
	}
	log.Printf("[DEBUG] total desired spot capacity: %f", totalDesiredSpotCapacity)

	desiredCapacity := InstanceCapacity{}
	for _, v := range availableVarieties {
		desiredCapacity[v] = totalDesiredSpotCapacity / float64(len(availableVarieties))
	}
	log.Printf("[INFO] desired capacity: %v", desiredCapacity)

	change := NewInstanceCapacityChange(spotCapacity, desiredCapacity)
	changeCount, err := change.Count()
	if err != nil {
		return err
	}
	log.Printf("[INFO] change count: %v", changeCount)

	if len(changeCount) == 0 {
		log.Println("[INFO] no change")
		return nil
	}

	err = r.confirmIfNeeded("")
	if err != nil {
		return err
	}

	eventDetails := []map[string]interface{}{}
	for v, c := range changeCount {
		eventDetails = append(eventDetails, map[string]interface{}{
			"count":   c,
			"variety": v,
		})
	}
	err = r.runHookCommands("scalingInstances", "Scaling instances", map[string]interface{}{
		"change": eventDetails,
	})
	if err != nil {
		return err
	}

	ami, err := r.config.AMICommand.Output([]string{})
	if err != nil {
		return err
	}

	err = r.takeCooldown()
	if err != nil {
		return err
	}

	for v, c := range changeCount {
		var err error
		err = r.confirmIfNeeded(fmt.Sprintf("%s * %d", v, c))
		if err != nil {
			return err
		}

		if c > 0 {
			err = r.updateTimer("LaunchingInstances")
			if err != nil {
				return err
			}

			err = r.ec2Client.LaunchInstances(v, c, ami)
			if err != nil {
				return err
			}
		} else if c < 0 {
			err = r.ec2Client.TerminateInstancesByCount(workingInstances.Managed(), v, c*-1)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Runner) takeCooldown() error {
	current, err := r.status.FetchCooldownEndsAt()
	if err != nil {
		return err
	}

	d, err := time.ParseDuration(r.config.Cooldown)
	if err != nil {
		return err
	}

	t := time.Now().Add(d)
	if t.After(current) {
		err := r.status.StoreCooldownEndsAt(t)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) correctScalingRate(scalingRate float64) float64 {
	if r.config.MaximumScalingRate > 0.0 && r.config.MaximumScalingRate < scalingRate {
		log.Printf("[INFO] computed scaling rate is %f but it is over MaximumScalingRate (%f)", scalingRate, r.config.MaximumScalingRate)
		scalingRate = r.config.MaximumScalingRate
	} else if r.config.MinimumScalingRate > scalingRate {
		log.Printf("[INFO] computed scaling rate is %f but it is under MinimumScalingRate (%f)", scalingRate, r.config.MinimumScalingRate)
		scalingRate = r.config.MinimumScalingRate
	}

	return scalingRate
}

func (r *Runner) correctDesiredTotalCapacity(totalCapacity float64) float64 {
	if r.config.MaximumCapacity > 0.0 && r.config.MaximumCapacity < totalCapacity {
		log.Printf("[INFO] computed desired capacity is %f but it is over MaximumCapacity (%f)", totalCapacity, r.config.MaximumCapacity)
		totalCapacity = r.config.MaximumCapacity
	} else if r.config.MinimumCapacity > totalCapacity {
		log.Printf("[INFO] computed desired capacity is %f but it is under MinimumCapacity (%f)", totalCapacity, r.config.MinimumCapacity)
		totalCapacity = r.config.MinimumCapacity
	}

	return totalCapacity
}

func (r *Runner) getCurrentSchedule() (*Schedule, error) {
	schedules, err := r.status.ListSchedules()
	if err != nil {
		return nil, err
	}

	for _, sch := range schedules {
		now := time.Now()
		if now.After(sch.StartAt) && now.Before(sch.EndAt) {
			return sch, nil
		}
	}
	return nil, nil
}

func (r *Runner) runHookCommands(event string, message string, detail interface{}) error {
	d := map[string]interface{}{
		"event":   event,
		"message": message,
		"detail":  detail,
	}
	input, err := json.Marshal(d)
	if err != nil {
		return err
	}

	for _, h := range r.config.HookCommands {
		err := h.RunWithStdin(string(input) + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) updateTimer(after string) error {
	for k, t := range r.config.Timers {
		if t.After == after {
			log.Println("[DEBUG] updating timer:", t)
			d, err := time.ParseDuration(t.Duration)
			if err != nil {
				return err
			}

			err = r.status.UpdateTimer(k, time.Now().Add(d))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) runExpiredTimers() error {
	keys, err := r.status.GetExpiredTimers()
	if err != nil {
		return err
	}
	for _, k := range keys {
		if t, ok := r.config.Timers[k]; ok {
			log.Println("[DEBUG] running timer command:", t)
			err := t.RunWithStdin("")
			if err != nil {
				return err
			}
		}
		err := r.status.DeleteTimer(k)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) confirmIfNeeded(msg string) error {
	if !r.config.ConfirmBeforeAction {
		return nil
	}

	if msg != "" {
		fmt.Println(msg)
	}
	fmt.Print("Are you sure to continue? (y/N): ")
	line, _, err := bufio.NewReader(os.Stdin).ReadLine()
	if err != nil {
		return fmt.Errorf("getting confirmation input from stdin failed")
	}

	if strings.HasPrefix(strings.ToLower(string(line)), "y") {
		return nil
	}
	return fmt.Errorf("canceled by user")
}
