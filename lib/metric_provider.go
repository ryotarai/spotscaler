package autoscaler

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sort"
	"time"
)

type Metric []float64

func (m Metric) Max() float64 {
	sort.Float64s(m)
	return m[len(m)-1]
}

func (m Metric) Median() float64 {
	sort.Float64s(m)
	return m[len(m)/2]
}

type MetricProvider interface {
	Values(instances Instances) (Metric, error)
}

func NewMetricProvider(t string, awsSession *session.Session) (MetricProvider, error) {
	switch t {
	case "CloudWatchEC2":
		return NewCloudWatchEC2MetricProvider(awsSession)
	default:
		return nil, fmt.Errorf("%s is invalid metric provider", t)
	}
}

type CloudWatchEC2MetricProvider struct {
	cloudwatch cloudwatchiface.CloudWatchAPI
}

func NewCloudWatchEC2MetricProvider(awsSession *session.Session) (*CloudWatchEC2MetricProvider, error) {
	return &CloudWatchEC2MetricProvider{
		cloudwatch: cloudwatch.New(awsSession),
	}, nil
}

func (p *CloudWatchEC2MetricProvider) Values(instances Instances) (Metric, error) {
	period := 60           // TODO
	duration := 5 * period // TODO: configurable

	utils := Metric{}
	for _, i := range instances {
		if *i.Monitoring.State != ec2.MonitoringStateEnabled {
			continue
		}

		params := &cloudwatch.GetMetricStatisticsInput{
			MetricName: aws.String("CPUUtilization"),
			Namespace:  aws.String("AWS/EC2"),
			Period:     aws.Int64(int64(period)),
			EndTime:    aws.Time(time.Now()),
			StartTime:  aws.Time(time.Now().Add(time.Duration(-1*duration) * time.Second)),
			Statistics: []*string{aws.String("Average")},
			Dimensions: []*cloudwatch.Dimension{
				{Name: aws.String("InstanceId"), Value: i.InstanceId},
			},
		}
		resp, err := p.cloudwatch.GetMetricStatistics(params)
		if err != nil {
			return nil, err
		}

		for _, p := range resp.Datapoints {
			utils = append(utils, *p.Average)
		}
	}

	return utils, nil
}
