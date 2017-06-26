package main

import (
	"fmt"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	cliGroup     = kingpin.Flag("group", "The Autoscaling group to update periodically").OverrideDefaultFromEnvar("GROUP").Default("").String()
	cliFrequency = kingpin.Flag("frequency", "How often to run the check").OverrideDefaultFromEnvar("FREQUENCY").Default("120s").Duration()
	cliExtras    = kingpin.Flag("extras", "Additional 'buffer' instances").OverrideDefaultFromEnvar("EXTRAS").Default("1").Int64()
	cliScaleDown = kingpin.Flag("scale-down-timeout", "How long to wait before scaling down (in minutes)").OverrideDefaultFromEnvar("SCALE_DOWN_TIMEOUT").Default("60").Float64()
	cliDryRun    = kingpin.Flag("dry", "Don't make any changes!").Bool()
)

func main() {
	kingpin.Parse()

	fmt.Println("Starting Autoscaler")

	if *cliDryRun {
		fmt.Println("Running in dry run mode")
	}

	// We use the ec2metadata service to determine the region of the ASGs.
	meta := ec2metadata.New(session.New(), &aws.Config{})
	region, err := meta.Region()
	if err != nil {
		panic(err)
	}

	var (
		svc       = autoscaling.New(session.New(&aws.Config{Region: aws.String(region)}))
		limiter   = time.Tick(*cliFrequency)
		prevScale = time.Now()
	)

	for {
		<-limiter

		// Creates the in-cluster config.
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}

		// Creates the clientset for querying APIs.
		k8s, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}

		fmt.Println("Looking up Autoscaling Group")

		asg, size, err := lookupASG(svc, region)
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("Calculating Deployments requests")

		cpu, mem, err := deploymentRequests(k8s)
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf("Kubernetes Deployments require the following amount to run CPU %d / Memory %d\n", cpu, mem)

		desired := int64(scaler(cpu, mem, size)) + *cliExtras

		fmt.Printf("The desired amount is: %d\n", desired)

		if desired < *asg.MinSize {
			fmt.Printf("The desired capacity (%d) is less than the ASG minimum constraint (%d)\n", desired, *asg.MinSize)
			desired = *asg.MinSize
		}

		if desired > *asg.MaxSize {
			fmt.Printf("The desired capacity (%d) is more than the ASG maximum constraint (%d)\n", desired, *asg.MaxSize)
			desired = *asg.MaxSize
		}

		if desired == *asg.DesiredCapacity {
			fmt.Printf("The desired capacity (%d) has not changed\n", *asg.DesiredCapacity)
			continue
		}

		// Check if this is a "down scale" event and if we have had one of these in the past X minutes.
		if desired < *asg.DesiredCapacity && time.Now().Sub(prevScale).Minutes() < *cliScaleDown {
			fmt.Println("Skipping this scale down event because: Cooling down")
			continue
		}

		fmt.Printf("Setting the desired capacity from %d to %d\n", *asg.DesiredCapacity, desired)

		// Don't make any changes. Perfect for debugging.
		if *cliDryRun {
			continue
		}

		_, err = svc.SetDesiredCapacity(&autoscaling.SetDesiredCapacityInput{
			AutoScalingGroupName: aws.String(*cliGroup),
			DesiredCapacity:      aws.Int64(desired),
		})
		if err != nil {
			fmt.Println(err)
		}

		prevScale = time.Now()
	}
}
