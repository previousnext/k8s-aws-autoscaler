package main

import (
	"fmt"
)

var InstanceTypes = []InstanceType{
	InstanceType{
		Name:   "t2.medium",
		CPU:    2000,
		Memory: 4000,
	},
}

type InstanceType struct {
	Name   string
	CPU    int
	Memory int
}

func getInstanceType(t string) (InstanceType, error) {
	for _, i := range InstanceTypes {
		if i.Name == t {
			return i, nil
		}
	}
	return InstanceType{}, fmt.Errorf("Failed to find instance type")
}
