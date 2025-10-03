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

// generateReport Orchestrates data collection and writes the markdown report
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

    // Workloads summary counts (cluster-wide)
    workloadsMD, err := GenerateWorkloadsInfoMarkdown(clientset)
    if err != nil {
        return fmt.Errorf("error generating workloads summary: %v", err)
    }

    // Ingresses overview (grouped by namespace)
    ingressMD, err := GenerateIngressesMarkdown(clientset)
    if err != nil {
        return fmt.Errorf("error generating ingresses section: %v", err)
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

    return saveReportToFile(clusterInfo, workloadsMD, nodes, totalCPU, totalMemory, ingressMD, resources, eventsSummary, imagesMD)
}

func getNodeMetrics(clientset *kubernetes.Clientset) ([]nodeInfoStruct, string, string, error) {
    ns, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return nil, "", "", fmt.Errorf("error getting nodes: %v", err)
    }

    var totalMilliCPU int64
    var totalMemBytes int64
    var list []nodeInfoStruct
    for _, n := range ns.Items {
        cpu := n.Status.Allocatable[corev1.ResourceCPU]
        mem := n.Status.Allocatable[corev1.ResourceMemory]
        info := n.Status.NodeInfo

        list = append(list, nodeInfoStruct{
            Name:     n.Name,
            CPU:      cpu.String(),
            Memory:   formatBytesHelper(mem.Value()),
            OSImage:  info.OSImage,
            Kubelet:  info.KubeletVersion,
            Provider: n.Spec.ProviderID,
        })

        totalMilliCPU += cpu.MilliValue()
        totalMemBytes += mem.Value()
    }

    totalCPU := fmt.Sprintf("%.2f vCPU", float64(totalMilliCPU)/1000.0)
    totalMemory := formatBytesHelper(totalMemBytes)
    return list, totalCPU, totalMemory, nil
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

// saveReportToFile renders the markdown and writes it to ~/a3k-reports/<timestamp>.md
func saveReportToFile(clusterInfo *ClusterInfo, workloadsMarkdown string, nodes []nodeInfoStruct, totalCPU, totalMemory string, ingressMarkdown string, resources []WorkloadResource, eventsSummary *EventSummary, imagesMarkdown string) error {
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
    clusterName := os.Getenv("A3K_CLUSTER_NAME")
    author := os.Getenv("A3K_AUTHOR")
    if author == "" {
        author = os.Getenv("USER")
        if author == "" {
            author = "N/A"
        }
    }
    title := "# Relatório do Cluster A3K ✨\n\n"
    if clusterName != "" {
        title = "# Relatório do Cluster " + clusterName + " ✨\n\n"
    }
    reportContent := title
    reportContent += "*Gerado em " + time.Now().Format(time.RFC1123) + "*\n\n"
    reportContent += "Gerado com: A3K\n\n"
    reportContent += "Autor: " + author + "\n\n"
    reportContent += "---\n\n"

    // Table of contents
    reportContent += "## Sumário\n\n"
    reportContent += "- [Visão Geral do Cluster](#visão-geral-do-cluster-)\n"
    reportContent += "- [Resumo de Workloads](#resumo-de-workloads-)\n"
    reportContent += "- [Detalhes dos Nodes](#detalhes-dos-nodes-)\n"
    reportContent += "- [Ingresses por Namespace](#ingresses-por-namespace-)\n"
    reportContent += "- [Análise de Recursos](#análise-de-recursos-)\n"
    reportContent += "- [Resumo de Eventos](#resumo-de-eventos-)\n"
    reportContent += "- [Auditoria de Imagens](#auditoria-de-imagens-bitnami-vs-bitnamilegacy)\n\n"

    // Cluster overview
    reportContent += "## Visão Geral do Cluster 🧭\n\n"
    reportContent += FormatClusterInfo(clusterInfo) + "\n"
    reportContent += "---\n\n"

    // Workloads summary (immediately after overview)
    if workloadsMarkdown != "" {
        reportContent += workloadsMarkdown + "\n"
        reportContent += "---\n\n"
    }

    // Node details (after workloads summary)
    reportContent += "## Detalhes dos Nodes 🖥️\n\n"
    reportContent += "Esta seção apresenta um panorama dos nodes que compõem o cluster, incluindo recursos de CPU e memória disponíveis, sistema operacional em uso, versão do kubelet e provedor. Essas informações permitem avaliar a capacidade total de infraestrutura, identificar a distribuição de workloads e apoiar decisões relacionadas a escalabilidade, manutenção e otimização do ambiente Kubernetes.\n\n"
    reportContent += "### Resumo de Recursos\n\n"
    reportContent += "| Recurso | Total |\n"
    reportContent += "| --- | --- |\n"
    reportContent += "| CPU | " + totalCPU + " |\n"
    reportContent += "| Memória | " + totalMemory + " |\n\n"
    reportContent += "### Lista de Nodes\n\n"
    reportContent += "| Nome | CPU | Memória | SO | Kubelet | Provedor |\n"
    reportContent += "| --- | --- | --- | --- | --- | --- |\n"
    for _, n := range nodes {
        reportContent += fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n", n.Name, n.CPU, n.Memory, n.OSImage, n.Kubelet, n.Provider)
    }
    reportContent += "---\n\n"

    // Ingresses section (immediately after nodes)
    if ingressMarkdown != "" {
        reportContent += ingressMarkdown + "\n"
        reportContent += "---\n\n"
    }

    // Resource analysis
    reportContent += "## Análise de Recursos 📦\n\n"
    reportContent += "Esta seção evidencia workloads que não possuem requests e/ou limits configurados.\n"
    reportContent += "Sem esses parâmetros, o cluster perde capacidade de prever consumo de recursos, o que pode resultar em alocação ineficiente e riscos de instabilidade.\n"
    reportContent += "A definição adequada de requests e limits é fundamental para garantir performance, previsibilidade de custos e resiliência da plataforma.\n\n"
    reportContent += "### Workloads sem Resource Requests/Limits\n"
    reportContent += FormatResourceTable(resources) + "\n"
    reportContent += "---\n\n"

    // Events summary
    reportContent += "## Resumo de Eventos 📣\n\n"
    reportContent += FormatEventSummaryMarkdown(eventsSummary)
    reportContent += "\n---\n\n"

    // Images audit
    if imagesMarkdown != "" {
        reportContent += imagesMarkdown + "\n"
        reportContent += "---\n\n"
    }

    if _, err := f.WriteString(reportContent); err != nil {
        return fmt.Errorf("error writing to report file: %v", err)
    }
    fmt.Printf("✅ Report generated successfully at: %s\n", reportFile)
    return nil
}
