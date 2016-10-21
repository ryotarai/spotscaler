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
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Runner struct {
	config     *Config
	status     StatusStoreIface
	awsSession *session.Session
	ec2Client  EC2ClientIface
}

func NewRunner(config *Config) (*Runner, error) {
	awsSess, err := session.NewSession(aws.NewConfig())
	if err != nil {
		return nil, err
	}

	runner := &Runner{
		config:     config,
		status:     NewStatusStore(config.RedisHost, config.RedisKeyPrefix),
		awsSession: awsSess,
		ec2Client:  NewEC2Client(ec2.New(awsSess), config),
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

	err = r.runExpiredTimers()
	if err != nil {
		return err
	}

	err = r.propagateSIRTagsToInstances()
	if err != nil {
		return err
	}

	err = r.cancelDeadSIRs()
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

func (r *Runner) cancelDeadSIRs() error {
	log.Println("[DEBUG] START: cancelDeadSIRs")

	sirs, err := r.ec2Client.DescribeDeadSIRs()
	if err != nil {
		return err
	}

	err = r.ec2Client.CancelOpenSIRs(sirs)
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

	worstTotalSpotCapacity := spotCapacity.TotalInWorstCase(r.config.AcceptableTermination)
	log.Printf("[DEBUG] in worst case, spot capacity change from %f to %f", spotCapacity.Total(), worstTotalSpotCapacity)

	cpuUtilToScaleOut := r.config.MaximumCPUUtil *
		(ondemandCapacity.Total() + worstTotalSpotCapacity) /
		(ondemandCapacity.Total() + spotCapacity.Total())
	cpuUtilToScaleIn := cpuUtilToScaleOut - r.config.ScaleInThreshold
	log.Printf("[DEBUG] cpu util to scale out: %f, cpu util to scale in: %f", cpuUtilToScaleOut, cpuUtilToScaleIn)

	cpuUtil, err := r.getCPUUtil()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] CPU util: %f", cpuUtil)

	err = r.status.StoreMetric(map[string]float64{
		"lastOndemandCapacity":        ondemandCapacity.Total(),
		"lastSpotCapacity":            spotCapacity.Total(),
		"lastAvailableVarieties":      float64(len(availableVarieties)),
		"lastUnavailableVarieties":    float64(len(price) - len(availableVarieties)),
		"lastSpotCapacityInWorstCase": worstTotalSpotCapacity,
		"lastCPUUtilToScaleOut":       cpuUtilToScaleOut,
		"lastCPUUtilToScaleIn":        cpuUtilToScaleIn,
		"lastCPUUtil":                 cpuUtil,
	})
	if err != nil {
		log.Printf("[WARN] storing metric failed")
	}

	cooldownEndsAt, err := r.status.FetchCooldownEndsAt()
	if err != nil {
		return err
	}

	if time.Now().Before(cooldownEndsAt) {
		log.Printf("[INFO] skip scaling in cooldown (it ends at %s)", cooldownEndsAt)
		return nil
	}

	if len(availableVarieties)-r.config.AcceptableTermination < 1 {
		log.Printf("[ERROR] available varieties are too few against acceptable termination (%d)", r.config.AcceptableTermination)
	}

	schedule, err := r.getCurrentSchedule()
	if err != nil {
		return err
	}

	var desiredCapacity InstanceCapacity
	if schedule == nil {
		if cpuUtil <= cpuUtilToScaleIn {
			log.Println("[DEBUG] scaling in")
		} else if cpuUtilToScaleOut <= cpuUtil {
			log.Println("[DEBUG] scaling out")
		} else {
			log.Println("[DEBUG] skip both scaling in and scaling out")
			return nil
		}

		desiredCapacity, err = DesiredCapacityFromTargetCPUUtil(
			availableVarieties,
			cpuUtil,
			r.config.MaximumCPUUtil,
			r.config.ScaleInThreshold/2.0,
			ondemandCapacity.Total(),
			spotCapacity.Total(),
			r.config.AcceptableTermination,
		)
		if err != nil {
			return err
		}
	} else {
		log.Println("[INFO] schedule found:", schedule)
		desiredCapacity, err = DesiredCapacityFromTotal(
			availableVarieties,
			schedule.Capacity-ondemandCapacity.Total(),
			r.config.AcceptableTermination,
		)
		if err != nil {
			return err
		}
	}

	log.Printf("[INFO] desired capacity: %v", desiredCapacity)

	if r.config.MaximumCapacity > 0 && desiredCapacity.Total() > r.config.MaximumCapacity {
		return fmt.Errorf("computed desired capacity is over MaximumCapacity %f", r.config.MaximumCapacity)
	}

	changeCount, err := spotCapacity.CountDiff(desiredCapacity)
	if err != nil {
		return err
	}

	for v, i := range changeCount {
		if schedule != nil && i < 0 {
			log.Printf("[WARN] with scheduled capacity, terminating an instance is not allowed: %v * %d", v, i)
			changeCount[v] = 0
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

	err = r.ec2Client.ChangeInstances(changeCount, ami, workingInstances.Managed())
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
