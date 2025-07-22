package main

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type nodeMetrics struct {
	cpu    resource.Quantity
	memory resource.Quantity
}

func getNodes(clientset *kubernetes.Clientset) error {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error getting nodes: %v", err)
	}

	// Initialize counters and maps
	totalNodes := len(nodes.Items)
	nodeInfo := make(map[string]nodeMetrics)
	machineTypes := make(map[string]int)
	var totalCPU, totalMemory resource.Quantity

	// Process each node
	for _, node := range nodes.Items {
		// Get machine type from node labels
		machineType := "unknown"
		if val, ok := node.Labels["node.kubernetes.io/instance-type"]; ok {
			machineType = val
		} else if val, ok := node.Labels["beta.kubernetes.io/instance-type"]; ok {
			machineType = val
		}
		machineTypes[machineType]++

		// Get node resources
		cpu := node.Status.Allocatable[corev1.ResourceCPU]
		memory := node.Status.Allocatable[corev1.ResourceMemory]

		totalCPU.Add(cpu)
		totalMemory.Add(memory)

		nodeInfo[node.Name] = nodeMetrics{
			cpu:    cpu,
			memory: memory,
		}
	}

	// Sort machine types by count
	type machineTypeCount struct {
		name  string
		count int
	}
	var sortedMachineTypes []machineTypeCount
	for name, count := range machineTypes {
		sortedMachineTypes = append(sortedMachineTypes, machineTypeCount{name, count})
	}
	sort.Slice(sortedMachineTypes, func(i, j int) bool {
		return sortedMachineTypes[i].count > sortedMachineTypes[j].count
	})

	// Print results
	fmt.Println("=== Nodes Overview ===")
	fmt.Printf("Total Nodes: %d\n\n", totalNodes)

	// Print machine types
	fmt.Println("Machine Types:")
	for _, mt := range sortedMachineTypes {
		fmt.Printf("  - %s: %d nodes\n", mt.name, mt.count)
	}

	// Print total resources
	fmt.Println("\nTotal Cluster Resources:")
	fmt.Printf("  CPU:     %s\n", totalCPU.String())
	fmt.Printf("  Memory:  %s\n", formatBytes(totalMemory.Value()))

	// Print per-node resources
	fmt.Println("\nNode Details:")
	for nodeName, metrics := range nodeInfo {
		fmt.Printf("\nNode: %s\n", nodeName)
		fmt.Printf("  CPU:     %s\n", metrics.cpu.String())
		fmt.Printf("  Memory:  %s\n", formatBytes(metrics.memory.Value()))
	}

	return nil
}

// formatBytes converts bytes to a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
