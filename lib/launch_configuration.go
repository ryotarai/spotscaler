package autoscaler

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

type LaunchConfiguration struct {
	KeyName                string               `yaml:"KeyName" validate:"required"`
	SecurityGroupIDs       []string             `yaml:"SecurityGroupIDs" validate:"required"`
	UserData               string               `yaml:"UserData"`
	IAMInstanceProfileName string               `yaml:"IAMInstanceProfileName"`
	BlockDeviceMappings    []BlockDeviceMapping `yaml:"BlockDeviceMappings"`
}

type BlockDeviceMapping struct {
	DeviceName  *string         `yaml:"DeviceName"`
	EBS         *EBSBlockDevice `yaml:"EBS"`
	NoDevice    *string         `yaml:"NoDevice"`
	VirtualName *string         `yaml:"VirtualName"`
}

type EBSBlockDevice struct {
	DeleteOnTermination *bool   `yaml:"DeleteOnTermination"`
	Encrypted           *bool   `yaml:"Encrypted"`
	IOPS                *int64  `yaml:"IOPS"`
	SnapshotID          *string `yaml:"SnapshotID"`
	VolumeSize          *int64  `yaml:"VolumeSize"`
	VolumeType          *string `yaml:"VolumeType"`
}

func (c LaunchConfiguration) SDKBlockDeviceMappings() []*ec2.BlockDeviceMapping {
	ret := []*ec2.BlockDeviceMapping{}
	for _, m := range c.BlockDeviceMappings {
		ret = append(ret, m.SDK())
	}
	return ret
}

func (m BlockDeviceMapping) SDK() *ec2.BlockDeviceMapping {
	return &ec2.BlockDeviceMapping{
		DeviceName:  m.DeviceName,
		NoDevice:    m.NoDevice,
		VirtualName: m.VirtualName,
		Ebs:         m.EBS.SDK(),
	}
}

func (e *EBSBlockDevice) SDK() *ec2.EbsBlockDevice {
	return &ec2.EbsBlockDevice{
		DeleteOnTermination: e.DeleteOnTermination,
		Encrypted:           e.Encrypted,
		SnapshotId:          e.SnapshotID,
		Iops:                e.IOPS,
		VolumeSize:          e.VolumeSize,
		VolumeType:          e.VolumeType,
	}
}
