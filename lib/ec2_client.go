package autoscaler

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"log"
	"strconv"
	"strings"
	"time"
)

type EC2ClientIface interface {
	TerminateInstancesByCount(instances Instances, v InstanceVariety, count int64) error
	TerminateInstances(instances Instances) error
	LaunchInstances(v InstanceVariety, c int64, ami string) error
	LaunchOndemandInstances(v InstanceVariety, c int64, ami string) error
	LaunchSpotInstances(v InstanceVariety, c int64, ami string) error
	DescribeWorkingInstances() (Instances, error)

	DescribePendingAndActiveSIRs() ([]*ec2.SpotInstanceRequest, error)
	PropagateTagsFromSIRsToInstances(reqs []*ec2.SpotInstanceRequest) error
	CreateStatusTagsOfSIRs(reqs []*ec2.SpotInstanceRequest, status string) error
	DescribeSpotPrices(vs []InstanceVariety) (map[InstanceVariety]float64, error)
	DescribeDeadSIRs() ([]*ec2.SpotInstanceRequest, error)
	CancelOpenSIRs(reqs []*ec2.SpotInstanceRequest) error
}

type EC2Client struct {
	ec2    ec2iface.EC2API
	config *Config
}

func NewEC2Client(ec2 ec2iface.EC2API, config *Config) *EC2Client {
	return &EC2Client{
		ec2:    ec2,
		config: config,
	}
}

func (c *EC2Client) TerminateInstancesByCount(instances Instances, v InstanceVariety, count int64) error {
	target := Instances{}
	for _, i := range instances {
		if count <= 0 {
			break
		}
		if i.Variety() == v {
			target = append(target, i)
			count -= 1
		}
	}

	return c.TerminateInstances(target)
}

func (c *EC2Client) TerminateInstances(instances Instances) error {
	ids := []*string{}
	for _, i := range instances {
		ids = append(ids, i.InstanceId)
	}

	params := &ec2.CreateTagsInput{
		DryRun:    aws.Bool(c.config.DryRun),
		Resources: ids,
		Tags:      c.config.TerminateTags.SDK(),
	}
	log.Printf("[DEBUG] terminating: %s", params)

	_, err := c.ec2.CreateTags(params)
	if err != nil {
		return err
	}

	return nil
}

func (c *EC2Client) LaunchInstances(v InstanceVariety, count int64, ami string) error {
	switch v.LaunchMethod {
	case "ondemand":
		return c.LaunchOndemandInstances(v, count, ami)
	case "spot":
		return c.LaunchSpotInstances(v, count, ami)
	}
	return fmt.Errorf("%s is invalid", v.LaunchMethod)
}

func (c *EC2Client) LaunchOndemandInstances(v InstanceVariety, count int64, ami string) error {
	securityGroupIds := []*string{}
	for _, i := range c.config.LaunchConfiguration.SecurityGroupIDs {
		securityGroupIds = append(securityGroupIds, aws.String(i))
	}

	userData := base64.StdEncoding.EncodeToString([]byte(c.config.LaunchConfiguration.UserData))

	params := &ec2.RunInstancesInput{
		DryRun:           aws.Bool(c.config.DryRun),
		ImageId:          aws.String(ami),
		MaxCount:         aws.Int64(count),
		MinCount:         aws.Int64(count),
		InstanceType:     aws.String(v.InstanceType),
		KeyName:          aws.String(c.config.LaunchConfiguration.KeyName),
		SecurityGroupIds: securityGroupIds,
		SubnetId:         aws.String(v.SubnetID),
		UserData:         aws.String(userData),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String(c.config.LaunchConfiguration.IAMInstanceProfileName),
		},
		InstanceInitiatedShutdownBehavior: aws.String(c.config.LaunchConfiguration.InstanceInitiatedShutdownBehavior),
		BlockDeviceMappings:               c.config.LaunchConfiguration.SDKBlockDeviceMappings(),
	}
	log.Printf("[INFO] launching ondemand instances: %s", params)

	reservation, err := c.ec2.RunInstances(params)
	if err != nil {
		return err
	}

	instanceIDs := []*string{}
	for _, i := range reservation.Instances {
		instanceIDs = append(instanceIDs, i.InstanceId)
	}

	capacity, err := v.Capacity()
	if err != nil {
		return err
	}

	tags := []*ec2.Tag{
		{Key: aws.String(c.config.CapacityTagKey), Value: aws.String(fmt.Sprint(capacity))},
		{Key: aws.String("ManagedBy"), Value: aws.String("spot-autoscaler")},
	}
	tags = append(tags, c.config.InstanceTags.SDK()...)

	createTagsParams := &ec2.CreateTagsInput{
		DryRun:    aws.Bool(c.config.DryRun),
		Resources: instanceIDs,
		Tags:      tags,
	}
	_, err = c.ec2.CreateTags(createTagsParams)
	if err != nil {
		return err
	}

	return nil
}

func (c *EC2Client) LaunchSpotInstances(v InstanceVariety, count int64, ami string) error {
	securityGroupIds := []*string{}
	for _, i := range c.config.LaunchConfiguration.SecurityGroupIDs {
		securityGroupIds = append(securityGroupIds, aws.String(i))
	}

	biddingPrice, ok := c.config.BiddingPriceByType[v.InstanceType]
	if !ok {
		return fmt.Errorf("Bidding price for %s is unknown", v.InstanceType)
	}

	userData := base64.StdEncoding.EncodeToString([]byte(c.config.LaunchConfiguration.UserData))

	requestSpotInstancesParams := &ec2.RequestSpotInstancesInput{
		DryRun:        aws.Bool(c.config.DryRun),
		SpotPrice:     aws.String(fmt.Sprintf("%f", biddingPrice)),
		InstanceCount: aws.Int64(count),
		LaunchSpecification: &ec2.RequestSpotLaunchSpecification{
			ImageId:          aws.String(ami),
			InstanceType:     aws.String(v.InstanceType),
			KeyName:          aws.String(c.config.LaunchConfiguration.KeyName),
			SecurityGroupIds: securityGroupIds,
			SubnetId:         aws.String(v.SubnetID),
			UserData:         aws.String(userData),
			IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
				Name: aws.String(c.config.LaunchConfiguration.IAMInstanceProfileName),
			},
			BlockDeviceMappings: c.config.LaunchConfiguration.SDKBlockDeviceMappings(),
		},
	}
	log.Printf("[INFO] requesting spot instances: %s", requestSpotInstancesParams)

	resp, err := c.ec2.RequestSpotInstances(requestSpotInstancesParams)
	if err != nil {
		return err
	}

	ids := []*string{}
	for _, req := range resp.SpotInstanceRequests {
		ids = append(ids, req.SpotInstanceRequestId)
	}

	capacity, err := v.Capacity()
	if err != nil {
		return err
	}

	tags := []*ec2.Tag{
		{Key: aws.String("RequestedBy"), Value: aws.String("spot-autoscaler")},
		{Key: aws.String("spot-autoscaler:Status"), Value: aws.String("pending")},
		{Key: aws.String(fmt.Sprintf("propagate:%s", c.config.CapacityTagKey)), Value: aws.String(fmt.Sprint(capacity))},
		{Key: aws.String("propagate:ManagedBy"), Value: aws.String("spot-autoscaler")},
	}
	for _, t := range c.config.InstanceTags {
		tags = append(tags, &ec2.Tag{Key: aws.String(fmt.Sprintf("propagate:%s", t.Key)), Value: aws.String(t.Value)})
	}

	createTagsParams := &ec2.CreateTagsInput{
		DryRun:    aws.Bool(c.config.DryRun),
		Resources: ids,
		Tags:      tags,
	}

	retry := 4
	for i := 0; i < retry; i++ {
		_, err = c.ec2.CreateTags(createTagsParams)
		if err == nil {
			break
		}
		if i < retry-1 {
			log.Printf("[INFO] CreateTags failed, will retry after 5 sec: %s", err)
			<-time.After(time.Duration(5) * time.Second)
		} else {
			return err
		}
	}

	return nil
}

func (c *EC2Client) DescribeWorkingInstances() (Instances, error) {
	filters := append(
		c.config.WorkingInstanceFilters.SDK(),
		&ec2.Filter{Name: aws.String("instance-state-name"), Values: []*string{aws.String("running")}},
	)
	params := &ec2.DescribeInstancesInput{
		Filters: filters,
	}
	instances := []*ec2.Instance{}
	err := c.ec2.DescribeInstancesPages(
		params,
		func(page *ec2.DescribeInstancesOutput, lastPage bool) bool {
			for _, res := range page.Reservations {
				instances = append(instances, res.Instances...)
			}
			return true
		})
	if err != nil {
		return nil, err
	}
	ret := Instances{}
	for _, i := range instances {
		ret = append(ret, NewInstanceFromSDK(i))
	}
	return ret, nil
}

func (c *EC2Client) DescribePendingAndActiveSIRs() ([]*ec2.SpotInstanceRequest, error) {
	params := &ec2.DescribeSpotInstanceRequestsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("active")},
			}, {
				Name:   aws.String("tag:RequestedBy"),
				Values: []*string{aws.String("spot-autoscaler")},
			}, {
				Name:   aws.String("tag:spot-autoscaler:Status"),
				Values: []*string{aws.String("pending")},
			},
		},
	}

	resp, err := c.ec2.DescribeSpotInstanceRequests(params)
	if err != nil {
		return nil, err
	}

	return resp.SpotInstanceRequests, nil
}

func (c *EC2Client) PropagateTagsFromSIRsToInstances(reqs []*ec2.SpotInstanceRequest) error {
	for _, req := range reqs {
		tags := []*ec2.Tag{}
		for _, t := range req.Tags {
			if strings.HasPrefix(*t.Key, "propagate:") {
				key := strings.TrimPrefix(*t.Key, "propagate:")
				tags = append(tags, &ec2.Tag{Key: &key, Value: t.Value})
			}
		}

		createTagsParams := &ec2.CreateTagsInput{
			DryRun:    aws.Bool(c.config.DryRun),
			Resources: []*string{req.InstanceId},
			Tags:      tags,
		}

		log.Printf("[DEBUG] CreateTags: %s", createTagsParams)
		_, err := c.ec2.CreateTags(createTagsParams)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *EC2Client) CreateStatusTagsOfSIRs(reqs []*ec2.SpotInstanceRequest, status string) error {
	ids := []*string{}

	for _, req := range reqs {
		ids = append(ids, req.SpotInstanceRequestId)
	}

	createTagsParams := &ec2.CreateTagsInput{
		DryRun:    aws.Bool(c.config.DryRun),
		Resources: ids,
		Tags: []*ec2.Tag{
			{Key: aws.String("spot-autoscaler:Status"), Value: aws.String(status)},
		},
	}

	log.Printf("[DEBUG] CreateTags: %s", createTagsParams)
	_, err := c.ec2.CreateTags(createTagsParams)
	if err != nil {
		return err
	}

	return nil
}

func (c *EC2Client) DescribeSpotPrices(vs []InstanceVariety) (map[InstanceVariety]float64, error) {
	res := map[InstanceVariety]float64{}

	varietiesByAZ := map[string][]InstanceVariety{}
	for _, v := range vs {
		varietiesByAZ[v.AvailabilityZone] = append(varietiesByAZ[v.AvailabilityZone], v)
	}

	for az, vs := range varietiesByAZ {
		instanceTypes := []*string{}
		for _, v := range vs {
			instanceTypes = append(instanceTypes, aws.String(v.InstanceType))
		}

		input := &ec2.DescribeSpotPriceHistoryInput{
			AvailabilityZone:    aws.String(az),
			InstanceTypes:       instanceTypes,
			ProductDescriptions: []*string{aws.String("Linux/UNIX (Amazon VPC)")}, // TODO: make configurable
		}

		found := map[InstanceVariety]bool{}
		var errInside error
		pageIndex := 1
		err := c.ec2.DescribeSpotPriceHistoryPages(input, func(page *ec2.DescribeSpotPriceHistoryOutput, lastPage bool) bool {
			log.Printf("[TRACE] DescribeSpotPriceHistory page %i", pageIndex)
			for _, v := range vs {
				if f := found[v]; f {
					// already found
					continue
				}

				latestTimestamp := time.Time{}
				latestPrice := 0.0
				for _, p := range page.SpotPriceHistory {
					if latestTimestamp.Before(*p.Timestamp) && *p.InstanceType == v.InstanceType && *p.AvailabilityZone == v.AvailabilityZone {
						latestTimestamp = *p.Timestamp
						f, err := strconv.ParseFloat(*p.SpotPrice, 64)
						if err != nil {
							errInside = err
							return false
						}

						latestPrice = f
					}
				}

				if latestPrice != 0.0 {
					// found
					res[v] = latestPrice
					found[v] = true
				}
			}

			pageIndex++
			return len(found) < len(vs)
		})

		if errInside != nil {
			return nil, errInside
		}

		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (c *EC2Client) DescribeDeadSIRs() ([]*ec2.SpotInstanceRequest, error) {
	params := &ec2.DescribeSpotInstanceRequestsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:RequestedBy"),
				Values: []*string{aws.String("spot-autoscaler")},
			}, {
				Name:   aws.String("state"),
				Values: []*string{aws.String("open")},
			},
		},
	}

	resp, err := c.ec2.DescribeSpotInstanceRequests(params)
	if err != nil {
		return nil, err
	}

	deadSIRs := []*ec2.SpotInstanceRequest{}
	for _, req := range resp.SpotInstanceRequests {
		if time.Now().Add(-5 * time.Minute).After(*req.CreateTime) {
			deadSIRs = append(deadSIRs, req)
		}
	}

	return deadSIRs, nil
}

func (c *EC2Client) CancelOpenSIRs(reqs []*ec2.SpotInstanceRequest) error {
	ids := []*string{}

	for _, req := range reqs {
		if *req.State == "open" {
			ids = append(ids, req.SpotInstanceRequestId)
		}
	}

	if len(ids) == 0 {
		return nil
	}

	cancelParams := &ec2.CancelSpotInstanceRequestsInput{
		DryRun:                 aws.Bool(c.config.DryRun),
		SpotInstanceRequestIds: ids,
	}
	log.Printf("[DEBUG] CancelSpotInstanceRequests: %s", cancelParams)
	_, err := c.ec2.CancelSpotInstanceRequests(cancelParams)
	if err != nil {
		return err
	}

	return nil
}
