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
    machineTypes := make(map[string]int)
    var totalCPU, totalMemory resource.Quantity
    var nodeRows [][]string
    var totalRequestedMilliCPU int64
    var totalRequestedBytesMemory int64

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

        // Additional node details
        info := node.Status.NodeInfo
        osImage := info.OSImage
        kubelet := info.KubeletVersion
        containerRuntime := info.ContainerRuntimeVersion
        arch := info.Architecture
        kernel := info.KernelVersion
        providerID := node.Spec.ProviderID

        // Topology details
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

        // IPs
        internalIP := ""
        externalIP := ""
        for _, addr := range node.Status.Addresses {
            if addr.Type == corev1.NodeInternalIP && internalIP == "" {
                internalIP = addr.Address
            } else if addr.Type == corev1.NodeExternalIP && externalIP == "" {
                externalIP = addr.Address
            }
        }

        // Aggregate requested resources and pod count per node (approximate consumption)
        var nodeRequestedMilliCPU int64
        var nodeRequestedBytesMemory int64
        var podCount int
        podsOnNode, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name)})
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

        // Node Ready condition
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

        nodeRows = append(nodeRows, []string{
            node.Name,
            internalIP,
            externalIP,
            machineType,
            cpu.String(),
            fmt.Sprintf("%.2f cores", float64(nodeRequestedMilliCPU)/1000.0),
            formatBytes(memory.Value()),
            formatBytes(nodeRequestedBytesMemory),
            osImage,
            kubelet,
            containerRuntime,
            arch,
            kernel,
            providerID,
            zone,
            region,
            fmt.Sprintf("%d", podCount),
            ready,
        })
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

    // Quick cluster analysis: per-node summary
    printHeader("Quick Cluster Analysis (Nodes)")
    quickRows := make([][]string, 0, len(nodeRows))
    for _, r := range nodeRows {
        // r indices: 0 Node,1 InternalIP,2 ExternalIP,3 Type,4 CPUAlloc,5 CPUReq,6 MemAlloc,7 MemReq, ... ,16 Pods,17 Ready
        quickRows = append(quickRows, []string{r[0], r[1], r[3], r[4], r[5], r[6], r[7], r[16], r[17]})
    }
    printTable([]string{"Node", "InternalIP", "Type", "CPU Alloc", "CPU Req", "Mem Alloc", "Mem Req", "Pods", "Ready"}, quickRows)

    // Concise totals summary
    printSubheader("Totals Summary")
    printTable([]string{"Metric", "Value"}, [][]string{
        {"Total Nodes", fmt.Sprintf("%d", totalNodes)},
        {"CPU Total", totalCPU.String()},
        {"CPU Requested", fmt.Sprintf("%.2f cores", float64(totalRequestedMilliCPU)/1000.0)},
        {"Memory Total", formatBytes(totalMemory.Value())},
        {"Memory Requested", formatBytes(totalRequestedBytesMemory)},
    })

    // Machine types summary (optional quick glance)
    printSubheader("Machine Types")
    mtRows := make([][]string, 0, len(sortedMachineTypes))
    for _, mt := range sortedMachineTypes {
        mtRows = append(mtRows, []string{mt.name, fmt.Sprintf("%d", mt.count)})
    }
    printTable([]string{"Type", "Nodes"}, mtRows)

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
