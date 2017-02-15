package ec2

import (
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
	"github.com/ryotarai/spotscaler/config"
)

type Client struct {
	ec2 SDKClient
	ui  cli.Ui
}

func NewClient(ui cli.Ui) (*Client, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	e := ec2.New(sess)

	return &Client{
		ec2: e,
		ui:  ui,
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

func (c *Client) DescribeCurrentSpotPrice(az string, instanceTypes []string) (map[string]float64, error) {
	types := []*string{}
	for _, t := range instanceTypes {
		types = append(types, aws.String(t))
	}

	input := &ec2.DescribeSpotPriceHistoryInput{
		AvailabilityZone:    aws.String(az),
		InstanceTypes:       types,
		ProductDescriptions: []*string{aws.String("Linux/UNIX (Amazon VPC)")},
	}

	result := map[string]float64{}
	foundByInstanceType := map[string]bool{}
	var errInside error
	err := c.ec2.DescribeSpotPriceHistoryPages(input, func(page *ec2.DescribeSpotPriceHistoryOutput, lastPage bool) bool {
		for _, instanceType := range instanceTypes {
			if foundByInstanceType[instanceType] {
				// already found
				continue
			}

			latestTimestamp := time.Time{}
			latestPrice := 0.0
			for _, p := range page.SpotPriceHistory {
				if latestTimestamp.Before(*p.Timestamp) && *p.InstanceType == instanceType {
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
				result[instanceType] = latestPrice
				foundByInstanceType[instanceType] = true
			}
		}

		return len(foundByInstanceType) < len(instanceTypes)
	})

	if errInside != nil {
		return nil, errInside
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}
