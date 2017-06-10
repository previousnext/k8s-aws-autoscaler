package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

// Step which calculates how much CPU + Memory is required to run all the pods on the cluster.
func deploymentRequests(k8s *kubernetes.Clientset) (int, int, error) {
	var (
		cpu int
		mem int
	)

	deployments, err := k8s.Extensions().Deployments(v1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return cpu, mem, err
	}

	for _, deployment := range deployments.Items {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			reqCPU := container.Resources.Requests[v1.ResourceCPU]
			reqMem := container.Resources.Requests[v1.ResourceMemory]

			cpu = cpu + int(reqCPU.MilliValue())*int(*deployment.Spec.Replicas)
			mem = mem + int(reqMem.Value()/1024.0/1024.0)*int(*deployment.Spec.Replicas)
		}
	}

	return cpu, mem, nil
}
