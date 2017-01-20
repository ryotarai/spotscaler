package ec2

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

type SDKClient interface {
	DescribeInstances(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
}
