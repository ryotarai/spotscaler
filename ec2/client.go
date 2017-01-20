package ec2

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/ryotarai/spotscaler/config"
)

type Client struct {
	ec2 SDKClient
}

func NewClient() (*Client, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	e := ec2.New(sess)

	return &Client{
		ec2: e,
	}, nil
}

func (c *Client) DescribeRunningInstances(filters []config.EC2Filter) (Instances, error) {
	fs := []*ec2.Filter{
		{Name: aws.String("instance-state-name"), Values: []*string{aws.String("running")}},
	}
	for _, f := range filters {
		values := []*string{}
		for _, v := range f.Values {
			values = append(values, aws.String(v))
		}
		fs = append(fs, &ec2.Filter{
			Name:   aws.String(f.Name),
			Values: values,
		})
	}

	params := &ec2.DescribeInstancesInput{
		Filters: fs,
	}
	instances := []*ec2.Instance{}

	for {
		resp, err := c.ec2.DescribeInstances(params)
		if err != nil {
			return nil, err
		}
		for _, res := range resp.Reservations {
			instances = append(instances, res.Instances...)
		}
		if resp.NextToken == nil {
			break
		}
		params.NextToken = resp.NextToken
	}

	ret := Instances{}
	for _, i := range instances {
		ret = append(ret, NewInstanceFromSDK(i))
	}
	return ret, nil
}
