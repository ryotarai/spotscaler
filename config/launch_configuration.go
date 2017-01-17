package config

type LaunchConfiguration struct {
	KeyName                string               `yaml:"KeyName" validate:"required"`
	SecurityGroupIDs       []string             `yaml:"SecurityGroupIDs" validate:"required"`
	UserData               string               `yaml:"UserData"`
	IAMInstanceProfileName string               `yaml:"IAMInstanceProfileName"`
	BlockDeviceMappings    []BlockDeviceMapping `yaml:"BlockDeviceMappings"`
	Subnets                []string             `yaml:"Subnets" validate:"required"`
}

type BlockDeviceMapping struct {
	DeviceName  string         `yaml:"DeviceName"`
	EBS         EBSBlockDevice `yaml:"EBS"`
	NoDevice    string         `yaml:"NoDevice"`
	VirtualName string         `yaml:"VirtualName"`
}

type EBSBlockDevice struct {
	DeleteOnTermination bool   `yaml:"DeleteOnTermination"`
	Encrypted           bool   `yaml:"Encrypted"`
	IOPS                int    `yaml:"IOPS"`
	SnapshotID          string `yaml:"SnapshotID"`
	VolumeSize          int    `yaml:"VolumeSize"`
	VolumeType          string `yaml:"VolumeType"`
}
