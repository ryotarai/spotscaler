package autoscaler

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

func indexOfStringInSlice(slice []string, target string) int {
	for i, s := range slice {
		if s == target {
			return i
		}
	}
	return -1
}

func findEC2TagByKey(tags []*ec2.Tag, key string) *ec2.Tag {
	for _, t := range tags {
		if *t.Key == key {
			return t
		}
	}

	return nil
}
