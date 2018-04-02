package autoscaler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Runner struct {
	config     *Config
	status     StatusStoreIface
	awsSession *session.Session
	ec2Client  EC2ClientIface
	api        *APIServer
}

func NewRunner(config *Config) (*Runner, error) {
	awsSess, err := session.NewSession(aws.NewConfig())
	if err != nil {
		return nil, err
	}

	status := NewStatusStore(config.RedisHost, config.FullAutoscalerID())

	runner := &Runner{
		config:     config,
		status:     status,
		awsSession: awsSess,
		ec2Client:  NewEC2Client(ec2.New(awsSess), config),
		api:        NewAPIServer(status),
	}

	return runner, nil
}

func (r *Runner) StartLoop() error {
	if r.config.APIAddr != "" {
		r.api.Run(r.config.APIAddr)
	}

	SetCapacityTable(r.config.InstanceCapacityByType)

	loopInterval, err := time.ParseDuration(r.config.LoopInterval)
	if err != nil {
		return err
	}

	for {
		c := time.After(loopInterval)

		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGTERM)

		if err != nil {
			log.Println("[ERROR] error in loop:", err)
		} else {
			err := r.Run()
			if err != nil {
				log.Println("[ERROR] error in loop:", err)
			}
		}

		select {
		case <-sigchan:
			log.Printf("[INFO] shutting down...")
			return nil
		default:
			signal.Stop(sigchan)
			log.Println("[INFO] waiting for next run")
			<-c
		}
	}

	return nil
}

func (r *Runner) Run() error {
	var err error

	log.Println("[DEBUG] START Runner.Run")
	if err != nil {
		return err
	}

	err = r.removeExpiredSchedules()
	if err != nil {
		return err
	}

	err = r.runExpiredTimers()
	if err != nil {
		return err
	}

	err = r.scale()
	if err != nil {
		return err
	}

	log.Println("[DEBUG] END Runner.Run")
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
	log.Printf("[DEBUG] spot capacity: %f", spotCapacity.Total())

	price, err := r.ec2Client.DescribeSpotPrices(r.config.InstanceVarieties())
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
	log.Printf("[DEBUG] %d spot varieties are available", len(availableVarieties))

	worstTotalSpotCapacity := spotCapacity.TotalInWorstCase(r.config.MaxTerminatedVarieties)
	log.Printf("[DEBUG] in worst case, spot capacity change from %f to %f", spotCapacity.Total(), worstTotalSpotCapacity)

	cpuUtilToScaleOut := r.config.MaxCPUUtil *
		(ondemandCapacity.Total() + worstTotalSpotCapacity) /
		(ondemandCapacity.Total() + spotCapacity.Total())
	cpuUtilToScaleIn := cpuUtilToScaleOut - r.config.ScaleInThreshold
	log.Printf("[DEBUG] cpu util to scale out: %f, cpu util to scale in: %f", cpuUtilToScaleOut, cpuUtilToScaleIn)

	cpuUtil, err := r.getCPUUtil()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] CPU util: %f", cpuUtil)

	r.api.UpdateMetrics(map[string]float64{
		"ondemand_capacity":           ondemandCapacity.Total(),
		"spot_capacity":               spotCapacity.Total(),
		"available_varieties":         float64(len(availableVarieties)),
		"unavailable_varieties":       float64(len(price) - len(availableVarieties)),
		"spot_capacity_in_worst_case": worstTotalSpotCapacity,
		"cpu_util_to_scale_out":       cpuUtilToScaleOut,
		"cpu_util_to_scale_in":        cpuUtilToScaleIn,
		"cpu_util":                    cpuUtil,
	})

	cooldownEndsAt, err := r.status.FetchCooldownEndsAt()
	if err != nil {
		return err
	}

	if time.Now().Before(cooldownEndsAt) {
		log.Printf("[INFO] skip scaling in cooldown (it ends at %s)", cooldownEndsAt)
		return nil
	}

	if len(availableVarieties)-r.config.MaxTerminatedVarieties < 1 {
		log.Printf("[ERROR] available varieties are too few against acceptable termination (%d)", r.config.MaxTerminatedVarieties)
	}

	schedule, err := r.getCurrentSchedule()
	if err != nil {
		return err
	}

	if schedule != nil {
		log.Printf("[INFO] schedule is found: %q", schedule)
	}

	var desiredCapacity InstanceCapacity
	if cpuUtil <= cpuUtilToScaleIn {
		log.Println("[DEBUG] scaling in")
	} else if cpuUtilToScaleOut <= cpuUtil {
		log.Println("[DEBUG] scaling out")
	} else if schedule == nil {
		log.Println("[DEBUG] skip both scaling in and scaling out")
		return nil
	}

	desiredCapacity, err = DesiredCapacityFromTargetCPUUtil(
		availableVarieties,
		cpuUtil,
		r.config.MaxCPUUtil,
		r.config.ScaleInThreshold/2.0,
		ondemandCapacity.Total(),
		spotCapacity.Total(),
		r.config.MaxTerminatedVarieties,
	)
	if err != nil {
		return err
	}

	if schedule != nil {
		log.Println("[INFO] schedule found:", schedule)
		dc, err := DesiredCapacityFromTotal(
			availableVarieties,
			schedule.Capacity-ondemandCapacity.Total(),
			r.config.MaxTerminatedVarieties,
		)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] capacity calculated from CPU util: %v", desiredCapacity)
		log.Printf("[DEBUG] capacity calculated from schedule: %v", dc)

		mtv := r.config.MaxTerminatedVarieties
		if dc.TotalInWorstCase(mtv) > desiredCapacity.TotalInWorstCase(mtv) {
			desiredCapacity = dc
		}
	}

	log.Printf("[INFO] desired capacity: %v", desiredCapacity)

	if r.config.MaxCapacity > 0 && desiredCapacity.Total() > r.config.MaxCapacity {
		return fmt.Errorf("computed desired capacity is over MaxCapacity %f", r.config.MaxCapacity)
	}

	if desiredCapacity.Total() <= r.config.MinCapacity {
		return fmt.Errorf("computed desired capacity is below MinCapacity %f <= %f", desiredCapacity.Total(), r.config.MinCapacity)
	}

	changeCount, err := spotCapacity.CountDiff(desiredCapacity)
	if err != nil {
		return err
	}

	prohibitToScaleIn := r.config.ProhibitToScaleIn
	if prohibitToScaleIn {
		log.Println("Scaling in is prohibited")
	}

	for v, i := range changeCount {
		if schedule != nil && i < 0 {
			log.Printf("[WARN] with scheduled capacity, terminating an instance is not allowed: %v * %d", v, i)
			delete(changeCount, v)
		} else if prohibitToScaleIn && i < 0 {
			log.Printf("[WARN] scaling in is prohibited, terminating an instance is not allowed: %v * %d", v, i)
			delete(changeCount, v)
		}
	}

	log.Printf("[INFO] change count: %v", changeCount)

	if len(changeCount) == 0 {
		log.Println("[INFO] no change")
		return nil
	}

	ami, err := r.config.AMICommand.Output([]string{})
	if err != nil {
		return err
	}

	if ami == "" {
		log.Println("[WARN] AMI is not found. Abort scaling activity")
		return nil
	}

	err = r.confirmIfNeeded("")
	if err != nil {
		return err
	}

	eventDetails := []map[string]interface{}{}
	for v, c := range changeCount {
		eventDetails = append(eventDetails, map[string]interface{}{
			"Count":   c,
			"Variety": v,
		})
	}
	err = r.runHookCommands("scalingInstances", "Scaling instances", map[string]interface{}{
		"Changes": eventDetails,
	})
	if err != nil {
		return err
	}

	err = r.takeCooldown()
	if err != nil {
		return err
	}

	err = r.ec2Client.ChangeInstances(changeCount, ami, workingInstances.ManagedBy(r.config.FullAutoscalerID()))
	if err != nil {
		return err
	}

	for _, c := range changeCount {
		if c > 0 {
			err = r.updateTimer("LaunchingInstances")
			if err != nil {
				return err
			}
			break
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

func (r *Runner) removeExpiredSchedules() error {
	schedules, err := r.status.ListSchedules()
	if err != nil {
		return err
	}

	now := time.Now()
	for _, sch := range schedules {
		if sch.EndAt.Before(now) {
			log.Printf("[INFO] Removing expired schedule: %s", sch.Key)
			err := r.status.RemoveSchedule(sch.Key)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Runner) getCurrentSchedule() (*Schedule, error) {
	schedules, err := r.status.ListSchedules()
	if err != nil {
		return nil, err
	}

	var activeSchedule *Schedule
	for _, sch := range schedules {
		now := time.Now()
		if now.After(sch.StartAt) && now.Before(sch.EndAt) {
			if activeSchedule == nil || activeSchedule.StartAt.Before(sch.StartAt) {
				activeSchedule = sch
			}
		}
	}

	return activeSchedule, nil
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

func (r *Runner) getCPUUtil() (float64, error) {
	s, err := r.config.CPUUtilCommand.Output([]string{})
	if err != nil {
		return 0.0, err
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0, err
	}

	return f, nil
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
