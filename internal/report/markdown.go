package report

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flysecurity/a3k/internal/k8s"
)

// Options controls report metadata.
type Options struct {
	ClusterName string
	Author      string
}

// Generate builds the full markdown report string from pre-collected data.
func Generate(
	clusterInfo *k8s.ClusterInfo,
	workloadsMD string,
	nodes *k8s.NodeSummary,
	ingressMD string,
	resources []k8s.WorkloadResource,
	healthMD string,
	securityMD string,
	eventsSummary *k8s.EventSummary,
	imagesMD string,
	opts Options,
) string {
	// Sanitize
	clusterName := sanitize(opts.ClusterName, 128)
	author := sanitize(opts.Author, 64)
	if author == "" {
		author = sanitize(os.Getenv("USER"), 64)
		if author == "" {
			author = "N/A"
		}
	}

	title := "# Relatório do Cluster A3K ✨\n\n"
	if clusterName != "" {
		title = "# Relatório do Cluster " + clusterName + " ✨\n\n"
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("*Gerado em " + time.Now().Format(time.RFC1123) + "*\n\n")
	sb.WriteString("Gerado com: A3K\n\n")
	sb.WriteString("Autor: " + author + "\n\n") //nolint:misspell // Portuguese
	sb.WriteString("---\n\n")

	// ToC
	sb.WriteString("## Sumário\n\n")
	sb.WriteString("- [Visão Geral do Cluster](#visão-geral-do-cluster-)\n")
	sb.WriteString("- [Resumo de Workloads](#resumo-de-workloads-)\n")
	sb.WriteString("- [Detalhes dos Nodes](#detalhes-dos-nodes-)\n")
	sb.WriteString("- [Ingresses por Namespace](#ingresses-por-namespace-)\n")
	sb.WriteString("- [Análise de Recursos](#análise-de-recursos-)\n")
	sb.WriteString("- [Saúde do Cluster](#saúde-do-cluster-)\n")
	sb.WriteString("- [Auditoria de Segurança](#auditoria-de-segurança-)\n")
	sb.WriteString("- [Resumo de Eventos](#resumo-de-eventos-)\n")
	sb.WriteString("- [Auditoria de Imagens](#auditoria-de-imagens-bitnami-vs-bitnamilegacy)\n\n")

	// Cluster overview
	sb.WriteString("## Visão Geral do Cluster 🧭\n\n")
	sb.WriteString(k8s.FormatClusterInfo(clusterInfo))
	sb.WriteString("\n---\n\n")

	// Workloads
	if workloadsMD != "" {
		sb.WriteString(workloadsMD + "\n---\n\n")
	}

	// Nodes
	sb.WriteString("## Detalhes dos Nodes 🖥️\n\n")
	sb.WriteString("Esta seção apresenta um panorama dos nodes que compõem o cluster, incluindo recursos de CPU e memória disponíveis, sistema operacional em uso, versão do kubelet e provedor. Essas informações permitem avaliar a capacidade total de infraestrutura, identificar a distribuição de workloads e apoiar decisões relacionadas a escalabilidade, manutenção e otimização do ambiente Kubernetes.\n\n") //nolint:misspell // Portuguese
	sb.WriteString("### Resumo de Recursos\n\n")
	sb.WriteString("| Recurso | Total |\n| --- | --- |\n")
	fmt.Fprintf(&sb, "| CPU | %s |\n", nodes.TotalCPU)
	fmt.Fprintf(&sb, "| Memória | %s |\n\n", nodes.TotalMemory)
	sb.WriteString("### Lista de Nodes\n\n")
	nodeRows := make([][]string, 0, len(nodes.Nodes))
	for _, n := range nodes.Nodes {
		nodeRows = append(nodeRows, []string{n.Name, n.CPUAllocatable, n.MemAllocatable, n.OSImage, n.KubeletVersion, n.ProviderID})
	}
	sb.WriteString(k8s.BuildMarkdownTable([]string{"Nome", "CPU", "Memória", "SO", "Kubelet", "Provedor"}, nodeRows))
	sb.WriteString("\n---\n\n")

	// Ingresses
	if ingressMD != "" {
		sb.WriteString(ingressMD + "\n---\n\n")
	}

	// Resources
	sb.WriteString("## Análise de Recursos 📦\n\n")
	sb.WriteString("Esta seção evidencia workloads que não possuem requests e/ou limits configurados.\n")
	sb.WriteString("Sem esses parâmetros, o cluster perde capacidade de prever consumo de recursos, o que pode resultar em alocação ineficiente e riscos de instabilidade.\n")
	sb.WriteString("A definição adequada de requests e limits é fundamental para garantir performance, previsibilidade de custos e resiliência da plataforma.\n\n")
	sb.WriteString("### Workloads sem Resource Requests/Limits\n")
	sb.WriteString(k8s.FormatResourceTable(resources) + "\n---\n\n")

	// Health
	if healthMD != "" {
		sb.WriteString(healthMD + "\n---\n\n")
	}

	// Security
	if securityMD != "" {
		sb.WriteString(securityMD + "\n---\n\n")
	}

	// Events
	sb.WriteString("## Resumo de Eventos 📣\n\n")
	sb.WriteString(k8s.FormatEventSummaryMarkdown(eventsSummary))
	sb.WriteString("\n---\n\n")

	// Images
	if imagesMD != "" {
		sb.WriteString(imagesMD + "\n---\n\n")
	}

	return sb.String()
}

func sanitize(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return strings.TrimSpace(s)
}
