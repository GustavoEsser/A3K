package main

import (
    "context"
    "fmt"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

func getWorkloads(clientset *kubernetes.Clientset) error {
	// Get all namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error getting namespaces: %v", err)
	}

	// Initialize counters
	var totalDeployments, totalStatefulSets, totalDaemonSets, totalCronJobs, totalPods int

	// Process each namespace
	for _, ns := range namespaces.Items {
		namespace := ns.Name

		// Count Deployments
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("error getting deployments in namespace %s: %v", namespace, err)
		}
		totalDeployments += len(deployments.Items)

		// Count StatefulSets
		statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("error getting statefulsets in namespace %s: %v", namespace, err)
		}
		totalStatefulSets += len(statefulSets.Items)

		// Count DaemonSets
		daemonSets, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("error getting daemonsets in namespace %s: %v", namespace, err)
		}
		totalDaemonSets += len(daemonSets.Items)

		// Count CronJobs
		cronJobs, err := clientset.BatchV1().CronJobs(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			// Skip if the API is not available (some clusters might not have batch/v1)
			if err.Error() != "the server could not find the requested resource" {
				return fmt.Errorf("error getting cronjobs in namespace %s: %v", namespace, err)
			}
		} else {
			totalCronJobs += len(cronJobs.Items)
		}

		// Count Running Pods
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{FieldSelector: "status.phase=Running"})
		if err != nil {
			return fmt.Errorf("error getting pods in namespace %s: %v", namespace, err)
		}
		totalPods += len(pods.Items)
	}

    // Print results (styled when gum is available)
    printHeader("Workloads Overview")
    printTable(
        []string{"Resource", "Count"},
        [][]string{
            {"Deployments", fmt.Sprintf("%d", totalDeployments)},
            {"StatefulSets", fmt.Sprintf("%d", totalStatefulSets)},
            {"DaemonSets", fmt.Sprintf("%d", totalDaemonSets)},
            {"CronJobs", fmt.Sprintf("%d", totalCronJobs)},
            {"Running Pods", fmt.Sprintf("%d", totalPods)},
        },
    )

	return nil
}
