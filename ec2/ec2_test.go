package ec2

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/ryotarai/spotscaler/mock"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestGetInstances(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sdk := mock.NewMockEC2API(ctrl)
	e := &EC2{
		sdk:            sdk,
		CapacityTagKey: "Capacity",
		WorkingFilters: map[string][]string{
			"tag:Status": []string{"working"},
		},
	}

	sdk.EXPECT().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("instance-state-name"), Values: aws.StringSlice([]string{"running"})},
			{Name: aws.String("tag:Status"), Values: aws.StringSlice([]string{"working"})},
		},
	}).Return(&ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			{Instances: []*ec2.Instance{
				{
					InstanceId:   aws.String("i-1"),
					InstanceType: aws.String("c4.large"),
					Placement: &ec2.Placement{
						AvailabilityZone: aws.String("ap-northeast-1b"),
					},
					Tags: []*ec2.Tag{{Key: aws.String("Capacity"), Value: aws.String("10.5")}},
				},
				{
					InstanceId:   aws.String("i-2"),
					InstanceType: aws.String("c4.large"),
					Placement: &ec2.Placement{
						AvailabilityZone: aws.String("ap-northeast-1b"),
					},
					Tags: []*ec2.Tag{{Key: aws.String("Capacity"), Value: aws.String("10.5")}},
					SpotInstanceRequestId: aws.String("sir-1"),
				},
			}},
		},
	}, nil)

	is, err := e.GetInstances()
	if assert.NoError(t, err) {
		assert.Len(t, is, 2)
		assert.Equal(t, *is[0], Instance{
			InstanceID:       "i-1",
			InstanceType:     "c4.large",
			AvailabilityZone: "ap-northeast-1b",
			Capacity:         10.5,
			Market:           "ondemand",
		})
		assert.Equal(t, *is[1], Instance{
			InstanceID:       "i-2",
			InstanceType:     "c4.large",
			AvailabilityZone: "ap-northeast-1b",
			Capacity:         10.5,
			Market:           "spot",
		})
	}
}

func TestLaunchInstances(t *testing.T) {
	logger, _ := logrustest.NewNullLogger()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sdk := mock.NewMockEC2API(ctrl)
	e := &EC2{
		sdk:            sdk,
		Logger:         logger,
		CapacityTagKey: "Capacity",
		SubnetByAZ: map[string]string{
			"az-1": "subnet-1",
		},
		KeyName:                "key",
		SecurityGroupIDs:       []string{"sg-1"},
		UserData:               "userdata",
		IAMInstanceProfileName: "profile",
		BlockDeviceMappings: []*BlockDeviceMapping{
			{
				DeviceName: "/dev/sda1",
				EBS: &BlockDeviceMappingEBS{
					DeleteOnTermination: true,
					VolumeSize:          10,
					VolumeType:          "gp2",
				},
			},
		},
	}

	in := &ec2.RunInstancesInput{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					VolumeSize:          aws.Int64(10),
					VolumeType:          aws.String("gp2"),
				},
			},
		},
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String("profile"),
		},
		ImageId: aws.String("dummy"),
		InstanceMarketOptions: &ec2.InstanceMarketOptionsRequest{
			MarketType: aws.String("spot"),
		},
		InstanceType:     aws.String("a"),
		KeyName:          aws.String("key"),
		MaxCount:         aws.Int64(1),
		MinCount:         aws.Int64(1),
		SecurityGroupIds: aws.StringSlice([]string{"sg-1"}),
		SubnetId:         aws.String("subnet-1"),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Capacity"),
						Value: aws.String("10.000000"),
					},
				},
			},
		},
		UserData: aws.String("dXNlcmRhdGE="),
	}
	sdk.EXPECT().RunInstances(in).Return(&ec2.Reservation{}, nil)

	e.LaunchInstances(Instances{
		{
			InstanceID:       "",
			InstanceType:     "a",
			AvailabilityZone: "az-1",
			Capacity:         10.0,
		},
	}, "dummy")
}

func TestTerminateInstances(t *testing.T) {
	logger, _ := logrustest.NewNullLogger()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sdk := mock.NewMockEC2API(ctrl)
	e := &EC2{
		sdk:    sdk,
		Logger: logger,
		TerminatingTag: map[string]string{
			"State": "terminating",
		},
	}

	sdk.EXPECT().CreateTags(&ec2.CreateTagsInput{
		Resources: aws.StringSlice([]string{"i-2"}),
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("State"),
				Value: aws.String("terminating"),
			},
		},
	})

	e.TerminateInstances(Instances{
		{InstanceID: "i-1"},
		{InstanceID: "i-2"},
	}, Instances{
		{InstanceID: "i-1"},
	})
}
