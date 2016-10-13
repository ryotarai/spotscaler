package autoscaler

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func configForTest() *Config {
	c := &Config{
		InstanceCapacityByType: map[string]float64{
			"c4.large": 10,
			"m4.large": 5,
		},
		FallbackInstanceVariety: InstanceVariety{
			InstanceType: "m4.large",
			SubnetID:     "subnet-abc",
			LaunchMethod: "ondemand",
		},
		InstanceVarieties: []InstanceVariety{
			{
				InstanceType: "c4.large",
				SubnetID:     "subnet-abc",
				LaunchMethod: "spot",
			},
			{
				InstanceType: "m4.large",
				SubnetID:     "subnet-abc",
				LaunchMethod: "spot",
			},
		},
		ScalingPolicies: []ScalingPolicy{
			{
				If:         "greaterThan",
				Threshold:  5,
				Target:     3,
				MetricType: "median",
			},
		},
		BiddingPriceByType: map[string]float64{
			"c4.large": 0.3,
			"m4.large": 0.3,
		},
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

func TestRecoverDeadSIRs(t *testing.T) {
	config := configForTest()
	reqs := []*ec2.SpotInstanceRequest{
		{
			SpotInstanceRequestId: aws.String("sir-abc"),
			LaunchSpecification: &ec2.LaunchSpecification{
				InstanceType: aws.String("c4.large"),
				SubnetId:     aws.String("subnet-dummy"),
			},
			Status: &ec2.SpotInstanceStatus{
				Code: aws.String("dummy"),
			},
		},
	}

	ec2Client := new(MockEC2ClientIface)
	ec2Client.On("DescribeDeadSIRs").Return(reqs, nil)
	ec2Client.On("LaunchInstances", config.FallbackInstanceVariety, int64(2), "ami-abc").Return(nil)
	ec2Client.On("CreateStatusTagsOfSIRs", reqs, "recovered").Return(nil)
	ec2Client.On("CancelOpenSIRs", reqs).Return(nil)

	r := &Runner{
		config:    config,
		ec2Client: ec2Client,
		ami:       "ami-abc",
	}
	recovered, err := r.recoverDeadSIRs()
	assert.True(t, recovered)
	assert.NoError(t, err)
	ec2Client.AssertExpectations(t)
}

func TestScale(t *testing.T) {
	config := configForTest()
	instance := Instance{
		Instance: ec2.Instance{
			InstanceId:   aws.String("i-abc"),
			InstanceType: aws.String("c4.large"),
			SubnetId:     aws.String("subnet-abc"),
		},
	}
	instances := Instances{
		&instance,
	}

	ec2Client := new(MockEC2ClientIface)
	ec2Client.On("DescribeWorkingInstances").Return(instances, nil)
	ec2Client.On("DescribeSpotPrices", config.InstanceVarieties).Return(map[InstanceVariety]float64{
		config.InstanceVarieties[0]: 0.1,
		config.InstanceVarieties[1]: 10, // too high
	}, nil)
	ec2Client.On("LaunchInstances", config.InstanceVarieties[0], int64(1), "ami-abc").Return(nil)
	ec2Client.On("LaunchInstances", config.FallbackInstanceVariety, int64(1), "ami-abc").Return(nil)

	statusStore := new(MockStatusStoreIface)
	statusStore.On("ListSchedules").Return([]*Schedule{}, nil)

	metricProvider := new(MockMetricProvider)
	metricProvider.On("Values", instances).Return([]float64{3, 4, 5, 6, 7, 8, 9}, nil)

	r := &Runner{
		config:         config,
		ec2Client:      ec2Client,
		ami:            "ami-abc",
		status:         statusStore,
		metricProvider: metricProvider,
	}
	scaled, err := r.scale()
	assert.True(t, scaled)
	assert.NoError(t, err)
	ec2Client.AssertExpectations(t)
}

func TestTerminateOndemand(t *testing.T) {
	config := configForTest()
	instances := Instances{
		{
			Instance: ec2.Instance{
				InstanceId:   aws.String("i-abc"),
				InstanceType: aws.String("c4.large"),
				SubnetId:     aws.String("subnet-abc"),
				Tags: []*ec2.Tag{
					{Key: aws.String("ManagedBy"), Value: aws.String("spot-autoscaler")},
				},
			},
		},
		{
			Instance: ec2.Instance{
				InstanceId:   aws.String("i-bcd"),
				InstanceType: aws.String("c4.large"),
				SubnetId:     aws.String("subnet-abc"),
				Tags: []*ec2.Tag{
					{Key: aws.String("ManagedBy"), Value: aws.String("spot-autoscaler")},
				},
			},
		},
	}

	ec2Client := new(MockEC2ClientIface)
	ec2Client.On("DescribeWorkingInstances").Return(instances, nil)
	ec2Client.On("TerminateInstances", Instances{instances[0]}).Return(nil)

	r := &Runner{
		config:    config,
		ec2Client: ec2Client,
		ami:       "ami-abc",
		lastTotalDesiredCapacity: 10.0,
	}
	err := r.terminateOndemand()
	assert.NoError(t, err)
	ec2Client.AssertExpectations(t)
}
