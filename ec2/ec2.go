package ec2

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/sirupsen/logrus"
)

type EC2 struct {
	sdk ec2iface.EC2API

	Logger                 *logrus.Logger
	DryRun                 bool
	WorkingFilters         map[string][]string
	CapacityTagKey         string
	SubnetByAZ             map[string]string
	KeyName                string
	SecurityGroupIDs       []string
	UserData               string
	IAMInstanceProfileName string
	BlockDeviceMappings    []*BlockDeviceMapping
}

type BlockDeviceMappingEBS struct {
	DeleteOnTermination bool
	VolumeSize          int64
	VolumeType          string
}

type BlockDeviceMapping struct {
	DeviceName string
	EBS        *BlockDeviceMappingEBS
}

func New() *EC2 {
	sess := session.Must(session.NewSession())
	return &EC2{
		sdk:            ec2.New(sess),
		WorkingFilters: map[string][]string{},
	}
}

func (e *EC2) GetInstances() (Instances, error) {
	filters := []*ec2.Filter{
		{Name: aws.String("instance-state-name"), Values: aws.StringSlice([]string{"running"})},
	}
	for n, v := range e.WorkingFilters {
		filters = append(filters, &ec2.Filter{Name: aws.String(n), Values: aws.StringSlice(v)})
	}
	output, err := e.sdk.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	instances := Instances{}
	for _, r := range output.Reservations {
		for _, i := range r.Instances {
			capStr, err := findTag(i.Tags, e.CapacityTagKey)
			if err != nil {
				return nil, err
			}

			cap, err := strconv.ParseFloat(capStr, 64)
			if err != nil {
				return nil, err
			}

			var market string
			if i.SpotInstanceRequestId == nil {
				market = "ondemand"
			} else {
				market = "spot"
			}

			instances = append(instances, &Instance{
				InstanceID:       *i.InstanceId,
				InstanceType:     *i.InstanceType,
				AvailabilityZone: *i.Placement.AvailabilityZone,
				Capacity:         cap,
				Market:           market,
			})
		}
	}

	return instances, nil
}

func (e *EC2) LaunchInstances(to Instances, ami string) error {
	count := map[string]int64{}
	for _, i := range to {
		if i.InstanceID != "" {
			break
		}

		count[fmt.Sprintf("%s/%s/%f", i.AvailabilityZone, i.InstanceType, i.Capacity)]++
	}

	userData := base64.StdEncoding.EncodeToString([]byte(e.UserData))

	for k, n := range count {
		a := strings.Split(k, "/")
		az := a[0]
		t := a[1]
		capacity := a[2]

		tags := []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{Key: aws.String(e.CapacityTagKey), Value: aws.String(capacity)},
				},
			},
		}

		in := &ec2.RunInstancesInput{
			SubnetId:          aws.String(e.SubnetByAZ[az]),
			InstanceType:      aws.String(t),
			MinCount:          aws.Int64(n),
			MaxCount:          aws.Int64(n),
			ImageId:           aws.String(ami),
			KeyName:           aws.String(e.KeyName),
			SecurityGroupIds:  aws.StringSlice(e.SecurityGroupIDs),
			UserData:          aws.String(userData),
			TagSpecifications: tags,
			InstanceMarketOptions: &ec2.InstanceMarketOptionsRequest{
				MarketType: aws.String("spot"),
			},
		}
		if e.IAMInstanceProfileName != "" {
			in.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
				Name: aws.String(e.IAMInstanceProfileName),
			}
		}
		if len(e.BlockDeviceMappings) > 0 {
			bdms := []*ec2.BlockDeviceMapping{}
			for _, b := range e.BlockDeviceMappings {
				bdm := &ec2.BlockDeviceMapping{
					DeviceName: aws.String(b.DeviceName),
					Ebs: &ec2.EbsBlockDevice{
						DeleteOnTermination: aws.Bool(b.EBS.DeleteOnTermination),
						VolumeSize:          aws.Int64(b.EBS.VolumeSize),
						VolumeType:          aws.String(b.EBS.VolumeType),
					},
				}
				bdms = append(bdms, bdm)
			}
			in.BlockDeviceMappings = bdms
		}

		if e.DryRun {
			e.Logger.Debugf("(dry run) Launching %#v", in)
			break
		}

		out, err := e.sdk.RunInstances(in)
		if err != nil {
			return err
		}

		for _, i := range out.Instances {
			e.Logger.Infof("Launched %s", *i.InstanceId)
		}
	}

	return nil
}

func (e *EC2) TerminateInstances(from, to Instances) error {
	toMap := map[string]*Instance{}

	for _, i := range to {
		if i.InstanceID != "" {
			toMap[i.InstanceID] = i
		}
	}

	for _, i := range from {
		if _, ok := toMap[i.InstanceID]; ok {
			break
		}

		if e.DryRun {
			e.Logger.Debugf("(dry run) Terminating an instance %s (%s in %s)", i.InstanceID, i.InstanceType, i.AvailabilityZone)
			break
		}
	}

	return nil
}

func findTag(tags []*ec2.Tag, key string) (string, error) {
	for _, t := range tags {
		if *t.Key == key {
			return *t.Value, nil
		}
	}

	return "", fmt.Errorf("%s tag is not found", key)
}
