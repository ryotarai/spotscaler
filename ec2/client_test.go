package ec2

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/ryotarai/spotscaler/config"
	"testing"
)

func TestDescribeRunningInstances(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockEC2API(ctrl)

	mock.EXPECT().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("instance-state-name"), Values: []*string{aws.String("running")}},
			{Name: aws.String("tag:Status"), Values: []*string{aws.String("working")}},
		},
	}).Return(&ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			{
				Instances: []*ec2.Instance{
					{
						Tags: []*ec2.Tag{
							{Key: aws.String("Status"), Value: aws.String("working")},
						},
					}, {
						Tags: []*ec2.Tag{
							{Key: aws.String("Status"), Value: aws.String("working")},
						},
					},
				},
			},
		},
	}, nil)

	c := &Client{
		ec2: mock,
	}

	instances, err := c.DescribeRunningInstances([]config.EC2Filter{
		{Name: "tag:Status", Values: []string{"working"}},
	})

	if err != nil {
		t.Error(err)
	}

	if len(instances) != 2 {
		t.Errorf("Got %d instances, wants %d", len(instances), 2)
	}
}
