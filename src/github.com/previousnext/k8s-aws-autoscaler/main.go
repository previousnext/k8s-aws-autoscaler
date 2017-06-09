package main

import (
	"log"
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
	cliGroup     = kingpin.Flag("group", "The Autoscaling group to update periodically").OverrideDefaultFromEnvar("GROUP").Default("").String()
	cliFrequency = kingpin.Flag("frequency", "How often to run the check").OverrideDefaultFromEnvar("FREQUENCY").Default("120s").Duration()
	cliBuffer    = kingpin.Flag("buffer", "Allows for hosts to have buffer eg. 80% full").OverrideDefaultFromEnvar("BUFFER").Default("0.9").Float64()
	cliDryRun    = kingpin.Flag("dry", "Don't make any changes!").Bool()
)

func main() {
	kingpin.Parse()

	log.Println("Starting Autoscaler")

	if *cliDryRun {
		log.Println("Running in dry run mode")
	}

	limiter := time.Tick(*cliFrequency)

	for {
		<-limiter

		// Creates the in-cluster config.
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Println(err)
			continue
		}

		// Creates the clientset for querying APIs.
		k8s, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Println(err)
			continue
		}

		// Build an Autoscaling client.
		// We use the ec2metadata service to determine the region of the ASGs.
		meta := ec2metadata.New(session.New(), &aws.Config{})
		region, err := meta.Region()
		if err != nil {
			log.Println(err)
			continue
		}
		svc := autoscaling.New(session.New(&aws.Config{Region: aws.String(region)}))

		// Query the ASGs based on the names provided.
		asgs, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{cliGroup},
			MaxRecords:            aws.Int64(int64(1)),
		})
		if err != nil {
			log.Println(err)
			continue
		}
		if len(asgs.AutoScalingGroups) != 1 {
			log.Println("Failed to lookup the ASG")
			continue
		}

		// This is the ASG we are looking for.
		asg := asgs.AutoScalingGroups[0]

		// Load up the pods and determine how much resource in total we require.
		pods, err := k8s.Pods(v1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			log.Println(err)
			continue
		}

		var (
			// These are the resource totals which we will be comparing against how many
			// machines we need to furfil the cluster.
			cpu int
			mem int
		)

		for _, pod := range pods.Items {
			// We only want to know about Pending and Running pods. These trigger scaling events.
			if pod.Status.Phase != v1.PodRunning || pod.Status.Phase != v1.PodPending {
				continue
			}

			for _, con := range pod.Spec.Containers {
				reqCPU := con.Resources.Requests[v1.ResourceCPU]
				reqMem := con.Resources.Requests[v1.ResourceMemory]

				cpu = cpu + int(reqCPU.MilliValue())
				mem = mem + int(reqMem.Value()/1024.0/1024.0)
			}
		}

		log.Printf("Kubernetes requires the following amount to run CPU %d / Memory %d", cpu, mem)

		// Determine the type of instance has been deployed for this ASG.
		// To get this information, we need to load the "Launch Configuration".
		cfgs, err := svc.DescribeLaunchConfigurations(&autoscaling.DescribeLaunchConfigurationsInput{
			LaunchConfigurationNames: []*string{
				asg.LaunchConfigurationName,
			},
			MaxRecords: aws.Int64(1),
		})
		if err != nil {
			log.Println(err)
			continue
		}
		if len(cfgs.LaunchConfigurations) != 1 {
			log.Println("Failed to lookup the ASG Launch Configuration:", *asg.LaunchConfigurationName)
			continue
		}

		// Query the instance type against our internal database eg. t2.medium = 2 CPUs + 4GB of Memory.
		inst, err := getInstanceType(*cfgs.LaunchConfigurations[0].InstanceType)
		if err != nil {
			log.Println("Failed to lookup the instance type:", *cfgs.LaunchConfigurations[0].InstanceType)
			continue
		}

		// Apply the buffer to each instance type.
		//   eg. 4000GB -> 3600GB (0.9)
		inst.CPU = int(float64(inst.CPU) * *cliBuffer)
		inst.Memory = int(float64(inst.Memory) * *cliBuffer)

		desired := int64(scaler(cpu, mem, inst))

		if desired < *asg.MinSize {
			log.Printf("The desired capacity (%d) is less than the ASG minimum constraint (%d)", desired, *asg.DesiredCapacity)
			desired = *asg.MinSize
		}

		if desired > *asg.MaxSize {
			log.Printf("The desired capacity (%d) is more than the ASG maximum constraint (%d)", desired, *asg.DesiredCapacity)
			desired = *asg.MaxSize
		}

		if desired == *asg.DesiredCapacity {
			log.Printf("The desired capacity (%d) has not changed", *asg.DesiredCapacity)
			continue
		}

		log.Printf("Setting the desired capacity from %d to %d", *asg.DesiredCapacity, desired)

		// Don't make any changes. Perfect for debugging.
		if *cliDryRun {
			continue
		}

		_, err = svc.SetDesiredCapacity(&autoscaling.SetDesiredCapacityInput{
			AutoScalingGroupName: aws.String(*cliGroup),
			DesiredCapacity:      aws.Int64(desired),
		})
		if err != nil {
			log.Println(err)
		}
	}
}
