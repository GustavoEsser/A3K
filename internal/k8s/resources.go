package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ResourceStatus represents the status of resource requests/limits for a container.
type ResourceStatus struct {
	HasRequests bool
	HasLimits   bool
}

// WorkloadResource represents a workload's resource configuration.
type WorkloadResource struct {
	Type      string
	Name      string
	Namespace string
	Status    string
}

// AnalyzeWorkloadResources analyzes all workloads and returns a list of those with missing requests/limits.
func AnalyzeWorkloadResources(clientset *kubernetes.Clientset) ([]WorkloadResource, error) {
	ctx := context.Background()
	var results []WorkloadResource

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		deployments, err := clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting deployments in namespace %s: %w", ns.Name, err)
		}
		for _, dep := range deployments.Items {
			status := checkContainers(dep.Spec.Template.Spec.Containers)
			if !status.HasRequests || !status.HasLimits {
				results = append(results, WorkloadResource{
					Type:      "Deployment",
					Name:      dep.Name,
					Namespace: ns.Name,
					Status:    getStatusEmoji(status),
				})
			}
		}

		statefulSets, err := clientset.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting statefulsets in namespace %s: %w", ns.Name, err)
		}
		for _, ss := range statefulSets.Items {
			status := checkContainers(ss.Spec.Template.Spec.Containers)
			if !status.HasRequests || !status.HasLimits {
				results = append(results, WorkloadResource{
					Type:      "StatefulSet",
					Name:      ss.Name,
					Namespace: ns.Name,
					Status:    getStatusEmoji(status),
				})
			}
		}

		daemonSets, err := clientset.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting daemonsets in namespace %s: %w", ns.Name, err)
		}
		for _, ds := range daemonSets.Items {
			status := checkContainers(ds.Spec.Template.Spec.Containers)
			if !status.HasRequests || !status.HasLimits {
				results = append(results, WorkloadResource{
					Type:      "DaemonSet",
					Name:      ds.Name,
					Namespace: ns.Name,
					Status:    getStatusEmoji(status),
				})
			}
		}

		cronJobs, err := clientset.BatchV1().CronJobs(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}
		for _, cj := range cronJobs.Items {
			status := checkContainers(cj.Spec.JobTemplate.Spec.Template.Spec.Containers)
			if !status.HasRequests || !status.HasLimits {
				results = append(results, WorkloadResource{
					Type:      "CronJob",
					Name:      cj.Name,
					Namespace: ns.Name,
					Status:    getStatusEmoji(status),
				})
			}
		}
	}

	return results, nil
}

// checkContainers checks all containers in a pod spec for resource requests/limits.
func checkContainers(containers []corev1.Container) ResourceStatus {
	status := ResourceStatus{HasRequests: true, HasLimits: true}
	for _, container := range containers {
		if container.Resources.Requests == nil || len(container.Resources.Requests) == 0 {
			status.HasRequests = false
		}
		if container.Resources.Limits == nil || len(container.Resources.Limits) == 0 {
			status.HasLimits = false
		}
		if !status.HasRequests || !status.HasLimits {
			break
		}
	}
	return status
}

// getStatusEmoji returns an emoji status based on the resource status.
func getStatusEmoji(status ResourceStatus) string {
	switch {
	case status.HasRequests && status.HasLimits:
		return "✅"
	case status.HasRequests:
		return "⚠️ (Faltando Limits)"
	case status.HasLimits:
		return "⚠️ (Faltando Requests)"
	default:
		return "❌ (Faltando Ambos)"
	}
}

// FormatResourceTable formats the workload resources as a markdown table.
func FormatResourceTable(resources []WorkloadResource) string {
	if len(resources) == 0 {
		return "✅ All workloads have proper resource requests and limits configured.\n"
	}

	table := "| Type | Name | Namespace | Status |\n"
	table += "|------|------|-----------|--------|\n"
	for _, r := range resources {
		table += fmt.Sprintf("| %s | %s | %s | %s |\n", r.Type, r.Name, r.Namespace, r.Status)
	}
	return table
}
