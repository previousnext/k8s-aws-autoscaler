package main

import (
	"fmt"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

var (
	cliGroup      = kingpin.Flag("group", "The Autoscaling group to update periodically").OverrideDefaultFromEnvar("GROUP").Default("").String()
	cliFrequency  = kingpin.Flag("frequency", "How often to run the check").OverrideDefaultFromEnvar("FREQUENCY").Default("120s").Duration()
	cliBuffer     = kingpin.Flag("buffer", "Allows for hosts to have buffer eg. 80% full").OverrideDefaultFromEnvar("BUFFER").Default("0.9").Float64()
	cliCronWeight = kingpin.Flag("cronjob-weight", "How much CronJobs should reflect autoscaling eg. 0.5 means provide half the resources").OverrideDefaultFromEnvar("CRONJOB_WEIGHT").Default("0.5").Float64()
	cliScaleDown  = kingpin.Flag("scale-down-timeout", "How long to wait before scaling down (in minutes)").OverrideDefaultFromEnvar("SCALE_DOWN_TIMEOUT").Default("60").Float64()
	cliDryRun     = kingpin.Flag("dry", "Don't make any changes!").Bool()
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
		svc      = autoscaling.New(session.New(&aws.Config{Region: aws.String(region)}))
		limiter  = time.Tick(*cliFrequency)
		lastDown = time.Now()
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

		// Apply the buffer to each instance type.
		//   eg. 4000GB -> 3600GB (0.9)
		size.CPU = int(float64(size.CPU) * *cliBuffer)
		size.Memory = int(float64(size.Memory) * *cliBuffer)

		fmt.Println("Calculating Pod requests")

		cpuPod, memPod, err := podRequests(k8s)
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("Calculating CronJob requests")

		cpuCron, memCron, err := cronRequests(k8s)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Apply the "what are CronJobs worth" weight.
		cpuCron = cpuCron * *cliCronWeight
		memCron = memCron * *cliCronWeight

		var (
			cpu = int(cpuPod + cpuCron)
			mem = int(memPod + memCron)
		)

		fmt.Printf("Kubernetes requires the following amount to run CPU %d / Memory %d", cpu, mem)

		desired := int64(scaler(cpu, mem, size))

		if desired < *asg.MinSize {
			fmt.Printf("The desired capacity (%d) is less than the ASG minimum constraint (%d)", desired, *asg.DesiredCapacity)
			desired = *asg.MinSize
		}

		if desired > *asg.MaxSize {
			fmt.Printf("The desired capacity (%d) is more than the ASG maximum constraint (%d)", desired, *asg.DesiredCapacity)
			desired = *asg.MaxSize
		}

		if desired == *asg.DesiredCapacity {
			fmt.Printf("The desired capacity (%d) has not changed", *asg.DesiredCapacity)
			continue
		}

		// Check if this is a "down scale" event and if we have had one of these in the past X minutes.
		if desired < *asg.DesiredCapacity && time.Now().Sub(lastDown).Minutes() > *cliScaleDown {
			fmt.Printf("Skipping this scale down event because cooldown is set to %d minutes", *cliScaleDown)
			continue
		}

		fmt.Printf("Setting the desired capacity from %d to %d", *asg.DesiredCapacity, desired)

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
	}
}

// Step which looks up the ASG and returns it.
func lookupASG(svc *autoscaling.AutoScaling, region string) (*autoscaling.Group, *InstanceType, error) {
	// Query the ASGs based on the names provided.
	asgs, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{cliGroup},
		MaxRecords:            aws.Int64(int64(1)),
	})
	if err != nil {
		return nil, nil, err
	}

	if len(asgs.AutoScalingGroups) != 1 {
		return nil, nil, fmt.Errorf("Failed to lookup the ASG")
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
		return nil, nil, err
	}

	if len(cfgs.LaunchConfigurations) != 1 {
		return nil, nil, fmt.Errorf("Failed to lookup the ASG Launch Configuration:", *asg.LaunchConfigurationName)
	}

	instanceType, err := getInstanceType(*cfgs.LaunchConfigurations[0].InstanceType)
	if err != nil {
		return nil, nil, err
	}

	return asg, instanceType, nil
}

// Step which calculates how much CPU + Memory is required to run all the pods on the cluster.
func podRequests(k8s *kubernetes.Clientset) (float64, float64, error) {
	var (
		cpu float64
		mem float64
	)

	pods, err := k8s.Pods(v1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return cpu, mem, err
	}

	for _, pod := range pods.Items {
		// We only want to know about Pending and Running pods. These trigger scaling events.
		if pod.Status.Phase != v1.PodRunning || pod.Status.Phase != v1.PodPending {
			continue
		}

		for _, con := range pod.Spec.Containers {
			reqCPU := con.Resources.Requests[v1.ResourceCPU]
			reqMem := con.Resources.Requests[v1.ResourceMemory]

			cpu = cpu + float64(reqCPU.MilliValue())
			mem = mem + float64(reqMem.Value()/1024.0/1024.0)
		}
	}

	return cpu, mem, nil
}

// Step which calculates how much CPU + Memory is required to run all the Cron Jobs on the cluster.
func cronRequests(k8s *kubernetes.Clientset) (float64, float64, error) {
	var (
		cpu float64
		mem float64
	)

	crons, err := k8s.BatchV2alpha1().CronJobs(v1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return cpu, mem, err
	}

	for _, cron := range crons.Items {
		// Specs all the way down!
		for _, container := range cron.Spec.JobTemplate.Spec.Template.Spec.Containers {
			reqCPU := container.Resources.Requests[v1.ResourceCPU]
			reqMem := container.Resources.Requests[v1.ResourceMemory]

			cpu = cpu + float64(reqCPU.MilliValue())
			mem = mem + float64(reqMem.Value()/1024.0/1024.0)
		}
	}

	return cpu, mem, nil
}
