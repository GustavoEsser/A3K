package k8s

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecurityFinding represents a security misconfiguration found in a workload.
type SecurityFinding struct {
	Severity  string
	Namespace string
	Kind      string
	Name      string
	Container string
	Issue     string
}

// SecurityAuditResult aggregates security findings from across the cluster.
type SecurityAuditResult struct {
	Findings []SecurityFinding
}

// AnalyzeSecurityPosture audits all workloads for common Kubernetes security misconfigurations.
func AnalyzeSecurityPosture(clientset *kubernetes.Clientset) (*SecurityAuditResult, error) {
	ctx := context.Background()
	result := &SecurityAuditResult{}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		if deps, err := clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
			for _, d := range deps.Items {
				result.Findings = append(result.Findings, auditPodSpec(ns.Name, "Deployment", d.Name, d.Spec.Template.Spec)...)
			}
		}
		if sss, err := clientset.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
			for _, s := range sss.Items {
				result.Findings = append(result.Findings, auditPodSpec(ns.Name, "StatefulSet", s.Name, s.Spec.Template.Spec)...)
			}
		}
		if dss, err := clientset.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
			for _, ds := range dss.Items {
				result.Findings = append(result.Findings, auditPodSpec(ns.Name, "DaemonSet", ds.Name, ds.Spec.Template.Spec)...)
			}
		}
		if cjs, err := clientset.BatchV1().CronJobs(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
			for _, cj := range cjs.Items {
				result.Findings = append(result.Findings, auditPodSpec(ns.Name, "CronJob", cj.Name, cj.Spec.JobTemplate.Spec.Template.Spec)...)
			}
		}
	}

	return result, nil
}

func auditPodSpec(namespace, kind, name string, spec corev1.PodSpec) []SecurityFinding {
	var findings []SecurityFinding

	if spec.HostNetwork {
		findings = append(findings, SecurityFinding{
			Severity: "HIGH", Namespace: namespace, Kind: kind, Name: name,
			Issue: "hostNetwork: true — compartilha o namespace de rede do node",
		})
	}
	if spec.HostPID {
		findings = append(findings, SecurityFinding{
			Severity: "HIGH", Namespace: namespace, Kind: kind, Name: name,
			Issue: "hostPID: true — compartilha o namespace de PID do node",
		})
	}
	if spec.HostIPC {
		findings = append(findings, SecurityFinding{
			Severity: "HIGH", Namespace: namespace, Kind: kind, Name: name,
			Issue: "hostIPC: true — compartilha o namespace de IPC do node",
		})
	}

	allContainers := append(spec.InitContainers, spec.Containers...) //nolint:gocritic
	for _, c := range allContainers {
		findings = append(findings, auditContainer(namespace, kind, name, c)...)
	}
	return findings
}

func auditContainer(namespace, kind, name string, c corev1.Container) []SecurityFinding {
	var findings []SecurityFinding
	sc := c.SecurityContext

	if sc == nil {
		findings = append(findings, SecurityFinding{
			Severity: "MEDIUM", Namespace: namespace, Kind: kind, Name: name, Container: c.Name,
			Issue: "securityContext não definido no container",
		})
		return findings
	}

	if sc.Privileged != nil && *sc.Privileged {
		findings = append(findings, SecurityFinding{
			Severity: "CRITICAL", Namespace: namespace, Kind: kind, Name: name, Container: c.Name,
			Issue: "privileged: true — container com acesso total ao host",
		})
	}

	if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
		findings = append(findings, SecurityFinding{
			Severity: "HIGH", Namespace: namespace, Kind: kind, Name: name, Container: c.Name,
			Issue: "runAsUser: 0 — container executa explicitamente como root",
		})
	}

	if sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot {
		findings = append(findings, SecurityFinding{
			Severity: "MEDIUM", Namespace: namespace, Kind: kind, Name: name, Container: c.Name,
			Issue: "runAsNonRoot não definido como true",
		})
	}

	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
		findings = append(findings, SecurityFinding{
			Severity: "MEDIUM", Namespace: namespace, Kind: kind, Name: name, Container: c.Name,
			Issue: "allowPrivilegeEscalation não definido como false",
		})
	}

	if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
		findings = append(findings, SecurityFinding{
			Severity: "LOW", Namespace: namespace, Kind: kind, Name: name, Container: c.Name,
			Issue: "readOnlyRootFilesystem não definido como true",
		})
	}

	return findings
}

// GenerateSecurityAuditMarkdown runs the security audit and returns a Markdown section.
func GenerateSecurityAuditMarkdown(clientset *kubernetes.Clientset) (string, error) {
	result, err := AnalyzeSecurityPosture(clientset)
	if err != nil {
		return "", fmt.Errorf("error analyzing security posture: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("## Auditoria de Segurança 🔒\n\n")
	sb.WriteString("Esta seção analisa a postura de segurança dos workloads do cluster, identificando configurações que podem expor o ambiente a riscos. As verificações cobrem contextos de segurança dos containers, permissões excessivas e exposição de namespaces do host.\n\n")

	if len(result.Findings) == 0 {
		sb.WriteString("✅ Nenhuma configuração de segurança problemática encontrada.\n")
		return sb.String(), nil
	}

	counts := map[string]int{"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0}
	for _, f := range result.Findings {
		counts[f.Severity]++
	}

	sb.WriteString("### Resumo por Severidade\n\n")
	sb.WriteString(BuildMarkdownTable([]string{"Severidade", "Quantidade"}, [][]string{
		{"🔴 CRITICAL", fmt.Sprintf("%d", counts["CRITICAL"])},
		{"🟠 HIGH", fmt.Sprintf("%d", counts["HIGH"])},
		{"🟡 MEDIUM", fmt.Sprintf("%d", counts["MEDIUM"])},
		{"🔵 LOW", fmt.Sprintf("%d", counts["LOW"])},
	}))
	sb.WriteString("\n")

	severityLabels := map[string]string{
		"CRITICAL": "🔴 CRITICAL",
		"HIGH":     "🟠 HIGH",
		"MEDIUM":   "🟡 MEDIUM",
		"LOW":      "🔵 LOW",
	}
	for _, severity := range []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"} {
		var rows [][]string
		for _, f := range result.Findings {
			if f.Severity == severity {
				rows = append(rows, []string{f.Namespace, f.Kind, f.Name, f.Container, f.Issue})
			}
		}
		if len(rows) > 0 {
			sb.WriteString(fmt.Sprintf("### %s\n\n", severityLabels[severity]))
			sb.WriteString(BuildMarkdownTable([]string{"Namespace", "Kind", "Name", "Container", "Problema"}, rows))
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
