package autoscaler

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func configForTest() *Config {
	c := &Config{
		Cooldown: "5m",
		AMICommand: Command{
			Command: "echo",
			Args:    []string{"-n", "ami-abc"},
		},
		InstanceCapacityByType: map[string]float64{
			"c4.large": 10,
			"m4.large": 10,
		},
		InstanceVarieties: []InstanceVariety{
			{
				InstanceType:     "c4.large",
				SubnetID:         "subnet-abc",
				LaunchMethod:     "spot",
				AvailabilityZone: "ap-northeast-1b",
			},
			{
				InstanceType:     "m4.large",
				SubnetID:         "subnet-abc",
				LaunchMethod:     "spot",
				AvailabilityZone: "ap-northeast-1b",
			},
			{
				InstanceType:     "r3.large",
				SubnetID:         "subnet-abc",
				LaunchMethod:     "spot",
				AvailabilityZone: "ap-northeast-1b",
			},
		},
		BiddingPriceByType: map[string]float64{
			"c4.large": 0.3,
			"m4.large": 0.3,
			"r3.large": 0.3,
		},
		AcceptableTermination:  1,
		RateOfCPUUtilToScaleIn: 0.5,
		MaximumCPUUtil:         80,
	}
	SetCapacityTable(c.InstanceCapacityByType)
	return c
}

func TestPropagateSIRTagsToInstances(t *testing.T) {
	reqs := []*ec2.SpotInstanceRequest{
		{SpotInstanceRequestId: aws.String("sir-abc")},
	}

	ec2Client := new(MockEC2ClientIface)
	ec2Client.On("DescribePendingAndActiveSIRs").Return(reqs, nil)
	ec2Client.On("PropagateTagsFromSIRsToInstances", reqs).Return(nil)
	ec2Client.On("CreateStatusTagsOfSIRs", reqs, "completed").Return(nil)

	r := &Runner{
		ec2Client: ec2Client,
	}
	err := r.propagateSIRTagsToInstances()
	assert.NoError(t, err)
	ec2Client.AssertExpectations(t)
}

func TestScaleOut(t *testing.T) {
	config := configForTest()
	instances := Instances{
		{
			Instance: ec2.Instance{
				InstanceId:            aws.String("i-abc"),
				InstanceType:          aws.String("c4.large"),
				SubnetId:              aws.String("subnet-abc"),
				SpotInstanceRequestId: nil, // ondemand
				Placement: &ec2.Placement{
					AvailabilityZone: aws.String("ap-northeast-1b"),
				},
			},
		},
		{
			Instance: ec2.Instance{
				InstanceId:            aws.String("i-abc"),
				InstanceType:          aws.String("c4.large"),
				SubnetId:              aws.String("subnet-abc"),
				SpotInstanceRequestId: aws.String("sir-abc"), // spot
				Placement: &ec2.Placement{
					AvailabilityZone: aws.String("ap-northeast-1b"),
				},
			},
		},
	}

	ec2Client := new(MockEC2ClientIface)
	ec2Client.On("DescribeWorkingInstances").Return(instances, nil)
	ec2Client.On("DescribeSpotPrices", config.InstanceVarieties).Return(map[InstanceVariety]float64{
		config.InstanceVarieties[0]: 0.1,
		config.InstanceVarieties[1]: 0.1,
		config.InstanceVarieties[2]: 10, // too high
	}, nil)
	ec2Client.On("LaunchInstances", config.InstanceVarieties[0], int64(2), "ami-abc").Return(nil)
	ec2Client.On("LaunchInstances", config.InstanceVarieties[1], int64(3), "ami-abc").Return(nil)

	statusStore := new(MockStatusStoreIface)
	statusStore.On("ListSchedules").Return([]*Schedule{}, nil)
	statusStore.On("FetchCooldownEndsAt").Return(time.Time{}, nil)
	statusStore.On("StoreCooldownEndsAt", mock.AnythingOfType("time.Time")).Return(nil)

	metricProvider := new(MockMetricProvider)
	metricProvider.On("Values", instances).Return(Metric{89, 89, 90, 91, 91}, nil)

	r := &Runner{
		config:         config,
		ec2Client:      ec2Client,
		status:         statusStore,
		metricProvider: metricProvider,
	}
	err := r.scale()
	assert.NoError(t, err)
	ec2Client.AssertExpectations(t)
}

func TestScaleIn(t *testing.T) {
	config := configForTest()
	instances := Instances{
		{
			Instance: ec2.Instance{
				InstanceId:            aws.String("i-1"),
				InstanceType:          aws.String("m4.large"),
				SubnetId:              aws.String("subnet-abc"),
				SpotInstanceRequestId: aws.String("sir-abc"), // spot
				Placement: &ec2.Placement{
					AvailabilityZone: aws.String("ap-northeast-1b"),
				},
				Tags: []*ec2.Tag{
					{Key: aws.String("ManagedBy"), Value: aws.String("spot-autoscaler")},
				},
			},
		},
		{
			Instance: ec2.Instance{
				InstanceId:            aws.String("i-2"),
				InstanceType:          aws.String("c4.large"),
				SubnetId:              aws.String("subnet-abc"),
				SpotInstanceRequestId: aws.String("sir-abc"), // spot
				Placement: &ec2.Placement{
					AvailabilityZone: aws.String("ap-northeast-1b"),
				},
				Tags: []*ec2.Tag{
					{Key: aws.String("ManagedBy"), Value: aws.String("spot-autoscaler")},
				},
			},
		},
		{
			Instance: ec2.Instance{
				InstanceId:            aws.String("i-3"),
				InstanceType:          aws.String("c4.large"),
				SubnetId:              aws.String("subnet-abc"),
				SpotInstanceRequestId: aws.String("sir-abc"), // spot
				Placement: &ec2.Placement{
					AvailabilityZone: aws.String("ap-northeast-1b"),
				},
				Tags: []*ec2.Tag{
					{Key: aws.String("ManagedBy"), Value: aws.String("spot-autoscaler")},
				},
			},
		},
	}

	ec2Client := new(MockEC2ClientIface)
	ec2Client.On("DescribeWorkingInstances").Return(instances, nil)
	ec2Client.On("DescribeSpotPrices", config.InstanceVarieties).Return(map[InstanceVariety]float64{
		config.InstanceVarieties[0]: 0.1,
		config.InstanceVarieties[1]: 0.1,
		config.InstanceVarieties[2]: 10, // too high
	}, nil)
	ec2Client.On("TerminateInstancesByCount", instances, config.InstanceVarieties[0], int64(1)).Return(nil)

	statusStore := new(MockStatusStoreIface)
	statusStore.On("ListSchedules").Return([]*Schedule{}, nil)
	statusStore.On("FetchCooldownEndsAt").Return(time.Time{}, nil)
	statusStore.On("StoreCooldownEndsAt", mock.AnythingOfType("time.Time")).Return(nil)

	metricProvider := new(MockMetricProvider)
	metricProvider.On("Values", instances).Return(Metric{5, 10}, nil)

	r := &Runner{
		config:         config,
		ec2Client:      ec2Client,
		status:         statusStore,
		metricProvider: metricProvider,
	}
	err := r.scale()
	assert.NoError(t, err)
	ec2Client.AssertExpectations(t)
}
