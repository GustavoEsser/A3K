package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type nodeInfoStruct struct {
	Name     string
	CPU      string
	Memory   string
	OSImage  string
	Kubelet  string
	Provider string
}

func generateReport(clientset *kubernetes.Clientset) error {
    // Get cluster information
    clusterInfo, err := GetClusterInfo(clientset)
    if err != nil {
        return fmt.Errorf("error getting cluster info: %v", err)
    }

	// Get node metrics
	nodes, totalCPU, totalMemory, err := getNodeMetrics(clientset)
	if err != nil {
		return fmt.Errorf("error getting node metrics: %v", err)
	}

    // Get resource analysis
    resources, err := AnalyzeWorkloadResources(clientset)
    if err != nil {
        return fmt.Errorf("error analyzing workload resources: %v", err)
    }

    // Get events summary
    eventsSummary, err := AnalyzeClusterEvents(clientset)
    if err != nil {
        return fmt.Errorf("error analyzing cluster events: %v", err)
    }

    // Get images audit markdown
    imagesMD, err := GenerateImagesAuditMarkdown(clientset)
    if err != nil {
        return fmt.Errorf("error generating images audit: %v", err)
    }

    // Save the report
    return saveReportToFile(clusterInfo, nodes, totalCPU, totalMemory, resources, eventsSummary, imagesMD)
}

func checkContainerResources(containers []corev1.Container, _, _, _ string) (bool, bool) {
	hasRequests := true
	hasLimits := true

	for _, container := range containers {
		if container.Resources.Requests == nil || container.Resources.Requests.Cpu().IsZero() || container.Resources.Requests.Memory().IsZero() {
			hasRequests = false
		}
		if container.Resources.Limits == nil || container.Resources.Limits.Cpu().IsZero() || container.Resources.Limits.Memory().IsZero() {
			hasLimits = false
		}
	}

	return hasRequests, hasLimits
}

func getWorkloadMetrics(clientset *kubernetes.Clientset) (int, int, int, int, int, error) {
	// Get Deployments
	deployments, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("error getting deployments: %v", err)
	}

	// Get StatefulSets
	statefulSets, err := clientset.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("error getting statefulsets: %v", err)
	}

	// Get DaemonSets
	daemonSets, err := clientset.AppsV1().DaemonSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("error getting daemonsets: %v", err)
	}

	// Get CronJobs
	cronJobs, err := clientset.BatchV1().CronJobs("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("error getting cronjobs: %v", err)
	}

	// Get Running Pods
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{FieldSelector: "status.phase=Running"})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("error getting pods: %v", err)
	}

	return len(deployments.Items), 
		len(statefulSets.Items), 
		len(daemonSets.Items), 
		len(cronJobs.Items), 
		len(pods.Items), 
		nil
}

func getNodeMetrics(clientset *kubernetes.Clientset) ([]nodeInfoStruct, string, string, error) {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, "", "", fmt.Errorf("error getting nodes: %v", err)
	}

	var totalCPU, totalMemory string
	var nodeList []nodeInfoStruct

	for _, node := range nodes.Items {
		// Get node resources
		cpu := node.Status.Allocatable[corev1.ResourceCPU]
		memory := node.Status.Allocatable[corev1.ResourceMemory]

		// Get node info
		var osImage, kubeletVersion, providerID string
		nodeInfo := node.Status.NodeInfo
		osImage = nodeInfo.OSImage
		kubeletVersion = nodeInfo.KubeletVersion

		providerID = node.Spec.ProviderID

		nodeList = append(nodeList, nodeInfoStruct{
			Name:     node.Name,
			CPU:      cpu.String(),
			Memory:   formatBytesHelper(memory.Value()),
			OSImage:  osImage,
			Kubelet:  kubeletVersion,
			Provider: providerID,
		})

		// Update total resources
		if totalCPU == "" {
			totalCPU = cpu.String()
			totalMemory = formatBytesHelper(memory.Value())
		} else {
			// This is a simplification - in a real scenario, you'd want to properly add quantities
			totalCPU = fmt.Sprintf("%s + %s", totalCPU, cpu.String())
			totalMemory = fmt.Sprintf("%s + %s", totalMemory, formatBytesHelper(memory.Value()))
		}
	}

	return nodeList, totalCPU, totalMemory, nil
}

func saveReportToFile(clusterInfo *ClusterInfo, nodes []nodeInfoStruct, totalCPU, totalMemory string, resources []WorkloadResource, eventsSummary *EventSummary, imagesMarkdown string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}

	reportDir := filepath.Join(homeDir, "a3k-reports")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("error creating reports directory: %v", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	reportFile := filepath.Join(reportDir, fmt.Sprintf("a3k-report-%s.md", timestamp))

	file, err := os.Create(reportFile)
	if err != nil {
		return fmt.Errorf("error creating report file: %v", err)
	}
	defer file.Close()
	// Start building the report content
	reportContent := "# A3K Cluster Report\n"
	reportContent += "*Generated on " + time.Now().Format(time.RFC1123) + "*\n\n"

	// Add cluster information
	reportContent += FormatClusterInfo(clusterInfo) + "\n"

	// Add resource analysis
	reportContent += "## Resource Analysis\n\n"
	reportContent += "### Workloads Missing Resource Requests/Limits\n"
	reportContent += FormatResourceTable(resources) + "\n"

	// Add events summary
	reportContent += FormatEventSummaryMarkdown(eventsSummary)

	// Add images audit
	if imagesMarkdown != "" {
		reportContent += "\n" + imagesMarkdown + "\n"
	}

	// Add node details
	reportContent += "## Node Details\n\n"
	reportContent += "### Resource Summary\n"
	reportContent += "- Total CPU: " + totalCPU + "\n"
	reportContent += "- Total Memory: " + totalMemory + "\n\n"

	reportContent += "### Node List\n"

	for _, node := range nodes {
		nodeInfo := fmt.Sprintf(
			"#### %s\n"+
			"- **CPU**: %s\n"+
			"- **Memory**: %s\n"+
			"- **OS**: %s\n"+
			"- **Kubelet**: %s\n"+
			"- **Provider**: %s\n\n",
			node.Name,
			node.CPU,
			node.Memory,
			node.OSImage,
			node.Kubelet,
			node.Provider,
		)
		reportContent += nodeInfo
	}

	// Write to file
	if _, err := file.WriteString(reportContent); err != nil {
		return fmt.Errorf("error writing to report file: %v", err)
	}

	fmt.Printf("✅ Report generated successfully at: %s\n", reportFile)
	return nil
}

// formatBytesHelper converts bytes to a human-readable string
func formatBytesHelper(bytes int64) string {
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
