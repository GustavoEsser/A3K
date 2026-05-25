package k8s

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NodeInfo holds per-node information.
type NodeInfo struct {
	Name             string
	InternalIP       string
	ExternalIP       string
	MachineType      string
	CPUAllocatable   string
	CPURequested     string
	MemAllocatable   string
	MemRequested     string
	OSImage          string
	KubeletVersion   string
	ContainerRuntime string
	Architecture     string
	KernelVersion    string
	ProviderID       string
	Zone             string
	Region           string
	PodCount         int
	Ready            string
}

// MachineTypeCount holds a machine type and its node count.
type MachineTypeCount struct {
	Name  string
	Count int
}

// NodeSummary holds cluster-wide node data.
type NodeSummary struct {
	Nodes                []NodeInfo
	TotalNodes           int
	TotalCPU             string
	TotalMemory          string
	TotalCPURequested    string
	TotalMemoryRequested string
	MachineTypes         []MachineTypeCount
}

// GetNodes fetches and returns node inventory and resource data.
func GetNodes(clientset *kubernetes.Clientset) (*NodeSummary, error) {
	ctx := context.Background()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting nodes: %w", err)
	}

	machineTypeCounts := make(map[string]int)
	var totalCPUMilli int64
	var totalMemBytes int64
	var totalRequestedMilliCPU int64
	var totalRequestedBytesMemory int64
	var nodeInfos []NodeInfo

	for _, node := range nodes.Items {
		machineType := "unknown"
		if val, ok := node.Labels["node.kubernetes.io/instance-type"]; ok {
			machineType = val
		} else if val, ok := node.Labels["beta.kubernetes.io/instance-type"]; ok {
			machineType = val
		}
		machineTypeCounts[machineType]++

		cpu := node.Status.Allocatable[corev1.ResourceCPU]
		memory := node.Status.Allocatable[corev1.ResourceMemory]
		totalCPUMilli += cpu.MilliValue()
		totalMemBytes += memory.Value()

		info := node.Status.NodeInfo
		providerID := node.Spec.ProviderID

		zone := "unknown"
		if val, ok := node.Labels["topology.kubernetes.io/zone"]; ok {
			zone = val
		} else if val, ok := node.Labels["failure-domain.beta.kubernetes.io/zone"]; ok {
			zone = val
		}

		region := "unknown"
		if val, ok := node.Labels["topology.kubernetes.io/region"]; ok {
			region = val
		} else if val, ok := node.Labels["failure-domain.beta.kubernetes.io/region"]; ok {
			region = val
		}

		internalIP := ""
		externalIP := ""
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP && internalIP == "" {
				internalIP = addr.Address
			} else if addr.Type == corev1.NodeExternalIP && externalIP == "" {
				externalIP = addr.Address
			}
		}

		var nodeRequestedMilliCPU int64
		var nodeRequestedBytesMemory int64
		var podCount int
		podsOnNode, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
		})
		if err == nil {
			podCount = len(podsOnNode.Items)
			for _, p := range podsOnNode.Items {
				for _, c := range p.Spec.Containers {
					if c.Resources.Requests != nil {
						if cpuReq, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
							nodeRequestedMilliCPU += cpuReq.MilliValue()
						}
						if memReq, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
							nodeRequestedBytesMemory += memReq.Value()
						}
					}
				}
			}
		}
		totalRequestedMilliCPU += nodeRequestedMilliCPU
		totalRequestedBytesMemory += nodeRequestedBytesMemory

		ready := "Unknown"
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				if cond.Status == corev1.ConditionTrue {
					ready = "Ready"
				} else {
					ready = string(cond.Status)
				}
				break
			}
		}

		nodeInfos = append(nodeInfos, NodeInfo{
			Name:             node.Name,
			InternalIP:       internalIP,
			ExternalIP:       externalIP,
			MachineType:      machineType,
			CPUAllocatable:   cpu.String(),
			CPURequested:     fmt.Sprintf("%.2f cores", float64(nodeRequestedMilliCPU)/1000.0),
			MemAllocatable:   FormatBytes(memory.Value()),
			MemRequested:     FormatBytes(nodeRequestedBytesMemory),
			OSImage:          info.OSImage,
			KubeletVersion:   info.KubeletVersion,
			ContainerRuntime: info.ContainerRuntimeVersion,
			Architecture:     info.Architecture,
			KernelVersion:    info.KernelVersion,
			ProviderID:       providerID,
			Zone:             zone,
			Region:           region,
			PodCount:         podCount,
			Ready:            ready,
		})
	}

	// Sort machine types by count descending
	var sortedMT []MachineTypeCount
	for name, count := range machineTypeCounts {
		sortedMT = append(sortedMT, MachineTypeCount{Name: name, Count: count})
	}
	sort.Slice(sortedMT, func(i, j int) bool {
		return sortedMT[i].Count > sortedMT[j].Count
	})

	return &NodeSummary{
		Nodes:                nodeInfos,
		TotalNodes:           len(nodes.Items),
		TotalCPU:             fmt.Sprintf("%.2f vCPU", float64(totalCPUMilli)/1000.0),
		TotalMemory:          FormatBytes(totalMemBytes),
		TotalCPURequested:    fmt.Sprintf("%.2f cores", float64(totalRequestedMilliCPU)/1000.0),
		TotalMemoryRequested: FormatBytes(totalRequestedBytesMemory),
		MachineTypes:         sortedMT,
	}, nil
}

// FormatBytes converts bytes to a human-readable string.
func FormatBytes(bytes int64) string {
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
