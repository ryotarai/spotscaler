package autoscaler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"math"
	"os"
	"strings"
	"time"
)

type Runner struct {
	config                   *Config
	status                   StatusStoreIface
	awsSession               *session.Session
	ec2Client                EC2ClientIface
	lastTotalDesiredCapacity float64
	ami                      string
	metricProvider           MetricProvider
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
		config:                   config,
		status:                   NewStatusStore(config.RedisHost, config.RedisKeyPrefix),
		awsSession:               awsSess,
		ec2Client:                NewEC2Client(ec2.New(awsSess), config),
		metricProvider:           metric,
		lastTotalDesiredCapacity: -1.0, // not set
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

	err = r.prepare()
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

	recovered, err := r.recoverDeadSIRs()
	if err != nil {
		return err
	}

	if recovered {
		err := r.takeCooldown()
		if err != nil {
			return err
		}
		return nil
	}

	cooldownEndsAt, err := r.status.FetchCooldownEndsAt()
	if err != nil {
		return err
	}

	if time.Now().Before(cooldownEndsAt) {
		log.Printf("[INFO] in cooldown (it ends at %s)", cooldownEndsAt)
		return nil
	}

	scaled, err := r.scale()
	if err != nil {
		return err
	}

	if scaled {
		err := r.takeCooldown()
		if err != nil {
			return err
		}
		return nil
	}

	err = r.terminateOndemand()
	if err != nil {
		return err
	}

	log.Println("[DEBUG] END Runner.Run")
	return nil
}

func (r *Runner) prepare() error {
	ami, err := r.findAMI()
	if err != nil {
		return err
	}
	r.ami = ami
	log.Println("[DEBUG] ami:", ami)

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

func (r *Runner) recoverDeadSIRs() (bool, error) {
	log.Println("[DEBUG] START: recoverDeadSIRs")

	// fetch SIRs whose status is not "recovered"
	deadSIRs, err := r.ec2Client.DescribeDeadSIRs()
	if err != nil {
		return false, err
	}

	if len(deadSIRs) == 0 {
		log.Println("[INFO] no dead spot instance request is found")
		return false, nil
	}

	log.Println("[INFO] dead spot instance requests are found")
	log.Printf("[DEBUG] dead spot instance requests: %s", deadSIRs)

	// notify hook
	details := []map[string]string{}
	for _, req := range deadSIRs {
		details = append(details, map[string]string{
			"spotInstanceRequestID": *req.SpotInstanceRequestId,
			"statusCode":            *req.Status.Code,
			"instanceType":          *req.LaunchSpecification.InstanceType,
			"subnetID":              *req.LaunchSpecification.SubnetId,
		})
	}
	err = r.runHookCommands("deadSpotInstanceRequests", "Some spot instance requests cannot be fulfilled or are terminated.", map[string]interface{}{
		"spotInstanceRequests": details,
	})
	if err != nil {
		return false, err
	}

	// compute total capacity of the instances
	totalCapacity := 0.0
	for _, req := range deadSIRs {
		c, err := CapacityFromInstanceType(*req.LaunchSpecification.InstanceType)
		if err != nil {
			return false, err
		}
		totalCapacity += c
	}
	log.Printf("[DEBUG] total capacity of dead requests: %f", totalCapacity)

	// launch ondemand instances which meet the capacity
	c, err := r.config.FallbackInstanceVariety.Capacity()
	if err != nil {
		return false, err
	}

	count := int64(math.Ceil(totalCapacity / c))
	log.Printf("[INFO] launching %s * %d", r.config.FallbackInstanceVariety, count)
	err = r.confirmIfNeeded("")
	if err != nil {
		return false, err
	}

	err = r.runHookCommands("scalingInstances", "Launching ondemand instances to recover spot instances", map[string]interface{}{
		"change": []map[string]interface{}{
			{
				"count":   count,
				"variety": r.config.FallbackInstanceVariety,
			},
		},
	})
	if err != nil {
		return false, err
	}

	err = r.updateTimer("LaunchingInstances")
	if err != nil {
		return false, err
	}

	err = r.ec2Client.LaunchInstances(r.config.FallbackInstanceVariety, count, r.ami)
	if err != nil {
		return false, err
	}

	// status:recovered tag to SIR
	err = r.ec2Client.CreateStatusTagsOfSIRs(deadSIRs, "recovered")
	if err != nil {
		return true, err
	}

	// cancel if open
	err = r.ec2Client.CancelOpenSIRs(deadSIRs)
	if err != nil {
		return true, err
	}

	return true, nil
}

func (r *Runner) scale() (bool, error) {
	log.Println("[DEBUG] START: scale")

	workingInstances, err := r.ec2Client.DescribeWorkingInstances()
	if err != nil {
		return false, err
	}

	totalDesiredCapacity, err := r.computeTotalDesiredCapacity(workingInstances)
	if err != nil {
		return false, err
	}

	if totalDesiredCapacity < 0 {
		log.Println("[INFO] scaling skipped")
		return false, nil
	}
	log.Printf("[INFO] totalDesiredCapacity: %f", totalDesiredCapacity)
	r.lastTotalDesiredCapacity = totalDesiredCapacity

	price, err := r.ec2Client.DescribeSpotPrices(r.config.InstanceVarieties)
	if err != nil {
		return false, err
	}

	availableVarieties := []InstanceVariety{}
	for v, p := range price {
		bid, ok := r.config.BiddingPriceByType[v.InstanceType]
		if !ok {
			return false, fmt.Errorf("Bidding price for %s is unknown", v.InstanceType)
		}

		if p <= bid {
			availableVarieties = append(availableVarieties, v)
		} else {
			log.Printf("[DEBUG] %v is not available due to price (%f USD)", v, p)
		}
	}

	totalSpotDesiredCapacity := totalDesiredCapacity * (float64(len(availableVarieties)) / float64(len(r.config.InstanceVarieties)))
	totalFallbackDesiredCapacity := totalDesiredCapacity - totalSpotDesiredCapacity

	desiredCapacity := InstanceCapacity{}
	for _, variety := range availableVarieties {
		desiredCapacity[variety] = totalSpotDesiredCapacity / float64(len(availableVarieties))
	}
	desiredCapacity[r.config.FallbackInstanceVariety] = totalFallbackDesiredCapacity
	log.Println("[DEBUG] desiredCapacity:", desiredCapacity)

	managedCapacity, err := InstanceCapacityFromInstances(workingInstances.Managed())
	if err != nil {
		return false, err
	}

	change := NewInstanceCapacityChange(managedCapacity, desiredCapacity)
	changeCount, err := change.Count()
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] change count: %s", changeCount)
	if len(changeCount) == 0 {
		return false, nil
	}

	err = r.confirmIfNeeded("")
	if err != nil {
		return false, err
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
		return false, err
	}

	for v, c := range changeCount {
		var err error
		err = r.confirmIfNeeded(fmt.Sprintf("%s * %d", v, c))
		if err != nil {
			return false, err
		}

		if c > 0 {
			err = r.updateTimer("LaunchingInstances")
			if err != nil {
				return true, err
			}

			err = r.ec2Client.LaunchInstances(v, c, r.ami)
			if err != nil {
				return true, err
			}
		} else if c < 0 {
			err = r.ec2Client.TerminateInstancesByCount(workingInstances.Managed(), v, c*-1)
			if err != nil {
				return true, err
			}
		}
	}

	return true, nil
}

func (r *Runner) computeTotalDesiredCapacity(instances Instances) (float64, error) {
	schedule, err := r.getCurrentSchedule()
	if err != nil {
		return -1.0, err
	}

	if schedule != nil {
		log.Println("[INFO] schedule found:", schedule)
		return schedule.Capacity, nil
	}

	values, err := r.metricProvider.Values(instances)
	if err != nil {
		return -1.0, err
	}
	log.Printf("[DEBUG] metric values: %s", values)

	scalingRate := 1.0
	for _, p := range r.config.ScalingPolicies {
		scalingRate, err = p.Rate(values)
		if err != nil {
			return -1.0, err
		}

		if scalingRate != 1.0 { // policy matched
			log.Println("[DEBUG] scalingRate:", scalingRate)
			scalingRate = r.correctScalingRate(scalingRate)
			break
		}
	}

	if scalingRate == 1.0 {
		// no policy matched
		return -1.0, nil
	}

	managedCapacity, err := InstanceCapacityFromInstances(instances.Managed())
	if err != nil {
		return -1.0, err
	}
	log.Println("[DEBUG] managedCapacity:", managedCapacity)

	workingCapacity, err := InstanceCapacityFromInstances(instances)
	if err != nil {
		return -1.0, err
	}
	log.Println("[DEBUG] workingCapacity:", workingCapacity)

	totalDesiredCapacity := (workingCapacity.Total() * scalingRate) - (workingCapacity.Total() - managedCapacity.Total())
	totalDesiredCapacity = r.correctDesiredTotalCapacity(totalDesiredCapacity)

	return totalDesiredCapacity, nil
}

func (r *Runner) terminateOndemand() error {
	log.Println("[DEBUG] START: terminateOndemand")
	if r.lastTotalDesiredCapacity < 0 {
		log.Println("[DEBUG] lastTotalDesiredCapacity is not set")
		return nil
	}

	workingInstances, err := r.ec2Client.DescribeWorkingInstances()
	if err != nil {
		return err
	}

	managed := workingInstances.Managed()
	managedCapacity, err := InstanceCapacityFromInstances(managed)
	if err != nil {
		return err
	}

	gap := managedCapacity.Total() - r.lastTotalDesiredCapacity
	if gap <= 0 {
		log.Println("[DEBUG] no ondemand instance will be terminated")
		return nil
	}
	log.Printf("[DEBUG] gap between managed capacity and desired capacity: %f", gap)

	termination := Instances{}
	for _, i := range managed.Ondemand() {
		cap, err := i.Variety().Capacity()
		if err != nil {
			return err
		}

		if gap >= cap {
			termination = append(termination, i)
			gap -= cap
		}
	}

	if len(termination) == 0 {
		log.Println("[DEBUG] no ondemand instance will be terminated")
		return nil
	}

	log.Printf("[INFO] the following ondemand instances will be terminated: %s", termination)
	err = r.confirmIfNeeded("")
	if err != nil {
		return err
	}

	err = r.ec2Client.TerminateInstances(termination)
	if err != nil {
		return err
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

func (r *Runner) findAMI() (string, error) {
	ami, err := r.config.AMICommand.Output([]string{})
	return ami, err
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
