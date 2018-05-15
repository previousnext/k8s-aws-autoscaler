package scaler

import (
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// WatchParams passed to the Watch function.
type WatchParams struct {
	// Group name of the autoscaling group.
	Group string
	// DryRun to ensure scaling events are correct.
	DryRun bool
	// Frequency of which to check for capacity changes.
	Frequency time.Duration
	// DownTimeout to wait before scaling down a cluster.
	DownTimeout float64
	// NodeCPU declare how much CPU a node has.
	NodeCPU int
	// NodeMemory declare how much memory a node has.
	NodeMemory int
}

// Watch for capacity changes and set the AWS autoscaling group desired state.
func Watch(w io.Writer, params WatchParams) error {
	if params.DryRun {
		fmt.Fprintln(w, "Running in dry run mode")
	}

	// We use the ec2metadata service to determine the region of the ASGs.
	meta := ec2metadata.New(session.New(), &aws.Config{})
	region, err := meta.Region()
	if err != nil {
		panic(err)
	}

	var (
		svc       = autoscaling.New(session.New(&aws.Config{Region: aws.String(region)}))
		limiter   = time.Tick(params.Frequency)
		prevScale = time.Now()
	)

	for {
		<-limiter

		// Creates the in-cluster config.
		config, err := rest.InClusterConfig()
		if err != nil {
			return err
		}

		// Creates the clientset for querying APIs.
		k8s, err := kubernetes.NewForConfig(config)
		if err != nil {
			return errors.Wrap(err, "")
		}

		fmt.Println("Looking up Autoscaling Group")

		asg, err := getScalingGroup(svc, params.Group, region)
		if err != nil {
			return err
		}

		fmt.Println("Calculating Deployments requests")

		cpu, mem, err := getDeploymentRequests(k8s)
		if err != nil {
			return err
		}

		fmt.Printf("Kubernetes Deployments require the following amount to run CPU %d / Memory %d\n", cpu, mem)

		desired := getDesired(cpu, mem, params.NodeCPU, params.NodeMemory)

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
		if desired < *asg.DesiredCapacity && time.Now().Sub(prevScale).Minutes() < params.DownTimeout {
			fmt.Println("Skipping this scale down event because: Cooling down")
			continue
		}

		fmt.Printf("Setting the desired capacity from %d to %d\n", *asg.DesiredCapacity, desired)

		// Don't make any changes. Perfect for debugging.
		if params.DryRun {
			continue
		}

		_, err = svc.SetDesiredCapacity(&autoscaling.SetDesiredCapacityInput{
			AutoScalingGroupName: aws.String(params.Group),
			DesiredCapacity:      aws.Int64(desired),
		})
		if err != nil {
			fmt.Println(err)
		}

		prevScale = time.Now()
	}
}

func getScalingGroup(svc *autoscaling.AutoScaling, name, region string) (*autoscaling.Group, error) {
	// Query the ASGs based on the names provided.
	asgs, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(name),
		},
		MaxRecords: aws.Int64(int64(1)),
	})
	if err != nil {
		return nil, err
	}

	if len(asgs.AutoScalingGroups) != 1 {
		return nil, errors.New("Failed to lookup the ASG")
	}

	return asgs.AutoScalingGroups[0], nil
}

// Helper function which calculates how much CPU + Memory is required to run all the deployments on the cluster.
func getDeploymentRequests(k8s *kubernetes.Clientset) (int, int, error) {
	var (
		cpu int
		mem int
	)

	deployments, err := k8s.Extensions().Deployments(corev1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return cpu, mem, err
	}

	for _, deployment := range deployments.Items {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			reqCPU := container.Resources.Requests[corev1.ResourceCPU]
			reqMem := container.Resources.Requests[corev1.ResourceMemory]

			cpu = cpu + int(reqCPU.MilliValue())*int(*deployment.Spec.Replicas)
			mem = mem + int(reqMem.Value()/1024.0/1024.0)*int(*deployment.Spec.Replicas)
		}
	}

	return cpu, mem, nil
}

// Helper function to determine desired instances for the autoscaling group.
func getDesired(requestsCPU, requestsMem, nodeCPU, nodeMemory int) int64 {
	var (
		desiredByCPU = requestsCPU / nodeCPU
		desiredByMem = requestsMem / nodeMemory
	)

	// We default to "desired by CPU" as our default.
	desired := desiredByCPU

	// If we have more instances because "desired by Memory" is higher, we use that instead.
	if desiredByMem > desiredByCPU {
		desired = desiredByMem
	}

	// We increase the "desired" by one because our division chops off a
	// certain percentage.
	//   eg. 2.45 = 2 (but we still need compute for the .45)
	desired++

	return int64(desired)
}
