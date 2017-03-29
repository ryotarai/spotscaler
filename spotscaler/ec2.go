package spotscaler

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2State struct {
	Instances []*Instance
}

type EC2 struct {
	sdk *ec2.EC2
}

func NewEC2() (*EC2, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	s := ec2.New(sess)
	return &EC2{
		sdk: s,
	}, nil
}

func (e *EC2) GetCurrentState(workingFilters map[string][]string) (*EC2State, error) {
	instances, err := e.getWorkingInstances(workingFilters)
	if err != nil {
		return nil, err
	}

	return &EC2State{
		Instances: instances,
	}, nil
}

func (e *EC2) getWorkingInstances(workingFilters map[string][]string) ([]*Instance, error) {
	filters := []*ec2.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("running")},
		},
	}
	for n, vs := range workingFilters {
		values := []*string{}
		for _, v := range vs {
			values = append(values, aws.String(v))
		}

		filters = append(filters, &ec2.Filter{
			Name:   aws.String(n),
			Values: values,
		})
	}

	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	output, err := e.sdk.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	is := []*Instance{}
	for _, r := range output.Reservations {
		for _, i := range r.Instances {
			is = append(is, e.newInstance(i))
		}
	}

	return is, nil
}

func (e *EC2) newInstance(i *ec2.Instance) *Instance {
	return NewInstance(*i.InstanceId)
}
