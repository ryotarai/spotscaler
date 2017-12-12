package ec2

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/ryotarai/spotscaler/mock"
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
