package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

// Step which looks up the ASG and returns it.
func lookupASG(svc *autoscaling.AutoScaling, region string) (*autoscaling.Group, InstanceType, error) {
	var instanceType InstanceType

	// Query the ASGs based on the names provided.
	asgs, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{cliGroup},
		MaxRecords:            aws.Int64(int64(1)),
	})
	if err != nil {
		return nil, instanceType, err
	}

	if len(asgs.AutoScalingGroups) != 1 {
		return nil, instanceType, fmt.Errorf("Failed to lookup the ASG")
	}

	asg := asgs.AutoScalingGroups[0]

	// Determine the type of instance has been deployed for this ASG.
	// To get this information, we need to load the "Launch Configuration".
	cfgs, err := svc.DescribeLaunchConfigurations(&autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{
			asg.LaunchConfigurationName,
		},
		MaxRecords: aws.Int64(1),
	})
	if err != nil {
		return nil, instanceType, err
	}

	if len(cfgs.LaunchConfigurations) != 1 {
		return nil, instanceType, fmt.Errorf("Failed to lookup the ASG Launch Configuration:", *asg.LaunchConfigurationName)
	}

	instanceType, err = getInstanceType(*cfgs.LaunchConfigurations[0].InstanceType)
	if err != nil {
		return nil, instanceType, err
	}

	return asg, instanceType, nil
}
