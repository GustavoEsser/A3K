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

// generateReport orchestrates data collection and writes the markdown report
func generateReport(clientset *kubernetes.Clientset) error {
	// Cluster info
	clusterInfo, err := GetClusterInfo(clientset)
	if err != nil {
		return fmt.Errorf("error getting cluster info: %v", err)
	}

	// Nodes summary
	nodes, totalCPU, totalMemory, err := getNodeMetrics(clientset)
	if err != nil {
		return fmt.Errorf("error getting node metrics: %v", err)
	}

	// Workload resource analysis
	resources, err := AnalyzeWorkloadResources(clientset)
	if err != nil {
		return fmt.Errorf("error analyzing workload resources: %v", err)
	}

	// Events summary
	eventsSummary, err := AnalyzeClusterEvents(clientset)
	if err != nil {
		return fmt.Errorf("error analyzing cluster events: %v", err)
	}

	// Images audit section (markdown)
	imagesMD, err := GenerateImagesAuditMarkdown(clientset)
	if err != nil {
		return fmt.Errorf("error generating images audit: %v", err)
	}

	return saveReportToFile(clusterInfo, nodes, totalCPU, totalMemory, resources, eventsSummary, imagesMD)
}

func getWorkloadMetrics(clientset *kubernetes.Clientset) (int, int, int, int, int, error) {
	deployments, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("error getting deployments: %v", err)
	}

	statefulSets, err := clientset.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("error getting statefulsets: %v", err)
	}

	daemonSets, err := clientset.AppsV1().DaemonSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("error getting daemonsets: %v", err)
	}

	cronJobs, err := clientset.BatchV1().CronJobs("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("error getting cronjobs: %v", err)
	}

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

	var totalMilliCPU int64
	var totalMemBytes int64
	var nodeList []nodeInfoStruct

	for _, node := range nodes.Items {
		cpu := node.Status.Allocatable[corev1.ResourceCPU]
		memory := node.Status.Allocatable[corev1.ResourceMemory]
		info := node.Status.NodeInfo

		nodeList = append(nodeList, nodeInfoStruct{
			Name:     node.Name,
			CPU:      cpu.String(),
			Memory:   formatBytesHelper(memory.Value()),
			OSImage:  info.OSImage,
			Kubelet:  info.KubeletVersion,
			Provider: node.Spec.ProviderID,
		})

		totalMilliCPU += cpu.MilliValue()
		totalMemBytes += memory.Value()
	}

	totalCPU := fmt.Sprintf("%.2f vCPU", float64(totalMilliCPU)/1000.0)
	totalMemory := formatBytesHelper(totalMemBytes)
	return nodeList, totalCPU, totalMemory, nil
}

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

// saveReportToFile renders the markdown and writes it to ~/a3k-reports/<timestamp>.md
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

	f, err := os.Create(reportFile)
	if err != nil {
		return fmt.Errorf("error creating report file: %v", err)
	}
	defer f.Close()

	// Header and metadata
	// Cluster name and author
	clusterName := os.Getenv("A3K_CLUSTER_NAME")
	author := os.Getenv("A3K_AUTHOR")
	if author == "" {
		// Fallback to system user if not provided
		author = os.Getenv("USER")
		if author == "" {
			author = "N/A"
		}
	}
	title := "# Relatório do Cluster A3K \n\n"
	if clusterName != "" {
		title = "# Relatório do Cluster " + clusterName + " ✨\n\n"
	}
	reportContent := title
	reportContent += "*Gerado em " + time.Now().Format(time.RFC1123) + "*\n\n"
	reportContent += "Gerado com ferramenta: A3K\n\n"
	reportContent += "Autor: " + author + "\n\n"
	reportContent += "---\n\n"
	reportContent += "## Sumário\n\n"
	reportContent += "- [Visão Geral do Cluster](#visão-geral-do-cluster-)\n"
	reportContent += "- [Análise de Recursos](#análise-de-recursos-)\n"
	reportContent += "- [Resumo de Eventos](#resumo-de-eventos-)\n"
	reportContent += "- [Auditoria de Imagens](#auditoria-de-imagens-bitnami-vs-bitnamilegacy)\n"
	reportContent += "- [Detalhes dos Nodes](#detalhes-dos-nodes-)\n\n"
	reportContent += "---\n\n"

	// Cluster Overview
	reportContent += "## Visão Geral do Cluster 🧭\n\n"
	reportContent += FormatClusterInfo(clusterInfo) + "\n"
	reportContent += "---\n\n"

	// Resource Analysis
	reportContent += "## Análise de Recursos 📦\n\n"
	reportContent += "> Esta seção destaca workloads sem requests/limits definidos.\n\n"
	reportContent += "### Workloads sem Resource Requests/Limits\n"
	reportContent += FormatResourceTable(resources) + "\n"
	reportContent += "---\n\n"

	// Events Summary
	reportContent += "## Resumo de Eventos 📣\n\n"
	reportContent += FormatEventSummaryMarkdown(eventsSummary)
	reportContent += "\n---\n\n"

	// Auditoria de Imagens
	if imagesMarkdown != "" {
		reportContent += imagesMarkdown + "\n"
		reportContent += "---\n\n"
	}

	// Detalhes dos Nodes
	reportContent += "## Detalhes dos Nodes 🖥️\n\n"
	reportContent += "### Resumo de Recursos\n\n"
	reportContent += "| Recurso | Total |\n"
	reportContent += "| --- | --- |\n"
	reportContent += "| CPU | " + totalCPU + " |\n"
	reportContent += "| Memória | " + totalMemory + " |\n\n"

	reportContent += "### Lista de Nodes\n\n"
	reportContent += "| Nome | CPU | Memória | SO | Kubelet | Provedor |\n"
	reportContent += "| --- | --- | --- | --- | --- | --- |\n"
	for _, node := range nodes {
		reportContent += fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			node.Name,
			node.CPU,
			node.Memory,
			node.OSImage,
			node.Kubelet,
			node.Provider,
		)
	}

	if _, err := f.WriteString(reportContent); err != nil {
		return fmt.Errorf("error writing to report file: %v", err)
	}
	fmt.Printf("✅ Report generated successfully at: %s\n", reportFile)
	return nil
}
