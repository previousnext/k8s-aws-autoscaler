package main

import (
	"fmt"
)

// InstanceType contains CPU and Memory values of EC2 instances.
type InstanceType struct {
	Name   string
	CPU    int
	Memory int
}

func getInstanceType(name string) (InstanceType, error) {
	instances := []InstanceType{
		// T2 Family.
		InstanceType{
			Name:   "t2.nano",
			CPU:    1000,
			Memory: 500,
		},
		InstanceType{
			Name:   "t2.micro",
			CPU:    1000,
			Memory: 1000,
		},
		InstanceType{
			Name:   "t2.small",
			CPU:    1000,
			Memory: 2000,
		},
		InstanceType{
			Name:   "t2.medium",
			CPU:    2000,
			Memory: 4000,
		},
		InstanceType{
			Name:   "t2.large",
			CPU:    2000,
			Memory: 8000,
		},
		InstanceType{
			Name:   "t2.xlarge",
			CPU:    4000,
			Memory: 16000,
		},
		InstanceType{
			Name:   "t2.xlarge",
			CPU:    8000,
			Memory: 32000,
		},

		// M4 Family.
		InstanceType{
			Name:   "m4.large",
			CPU:    2000,
			Memory: 8000,
		},
		InstanceType{
			Name:   "m4.xlarge",
			CPU:    4000,
			Memory: 16000,
		},
		InstanceType{
			Name:   "m4.2xlarge",
			CPU:    8000,
			Memory: 32000,
		},
		InstanceType{
			Name:   "m4.4xlarge",
			CPU:    16000,
			Memory: 364000,
		},
		InstanceType{
			Name:   "m4.10xlarge",
			CPU:    40000,
			Memory: 64000,
		},
		InstanceType{
			Name:   "m4.16xlarge",
			CPU:    64000,
			Memory: 256000,
		},
		// M3 Family.
		InstanceType{
			Name:   "m3.medium",
			CPU:    1000,
			Memory: 3750,
		},
		InstanceType{
			Name:   "m3.large",
			CPU:    2000,
			Memory: 7500,
		},
		InstanceType{
			Name:   "m3.xlarge",
			CPU:    4000,
			Memory: 15000,
		},
		InstanceType{
			Name:   "m3.2xlarge",
			CPU:    8000,
			Memory: 30000,
		},
	}

	for _, instance := range instances {
		if instance.Name == name {
			return instance, nil
		}
	}
	return InstanceType{}, fmt.Errorf("Failed to find instance type")
}
