package k8s

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ImageEntry holds details about a container image finding.
type ImageEntry struct {
	Namespace string
	Kind      string
	Name      string
	Container string
	Image     string
}

// ImageAuditResult holds the results of the images audit.
type ImageAuditResult struct {
	BitnamiControllers []ImageEntry
	LegacyControllers  []ImageEntry
	BitnamiPods        []ImageEntry
	LegacyPods         []ImageEntry
}

// AuditImages aggregates controllers and pods using Bitnami/BitnamiLegacy images.
func AuditImages(clientset *kubernetes.Clientset) (*ImageAuditResult, error) {
	ctx := context.Background()
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting namespaces: %w", err)
	}

	result := &ImageAuditResult{}

	for _, ns := range namespaces.Items {
		if deps, err := clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
			for _, d := range deps.Items {
				addControllerImages(ns.Name, "Deployment", d.Name, d.Spec.Template.Spec, &result.BitnamiControllers, &result.LegacyControllers)
			}
		}
		if sss, err := clientset.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
			for _, s := range sss.Items {
				addControllerImages(ns.Name, "StatefulSet", s.Name, s.Spec.Template.Spec, &result.BitnamiControllers, &result.LegacyControllers)
			}
		}
		if dss, err := clientset.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
			for _, ds := range dss.Items {
				addControllerImages(ns.Name, "DaemonSet", ds.Name, ds.Spec.Template.Spec, &result.BitnamiControllers, &result.LegacyControllers)
			}
		}
		if cjs, err := clientset.BatchV1().CronJobs(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
			for _, cj := range cjs.Items {
				addControllerImages(ns.Name, "CronJob", cj.Name, cj.Spec.JobTemplate.Spec.Template.Spec, &result.BitnamiControllers, &result.LegacyControllers)
			}
		}
		if pods, err := clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
			for _, p := range pods.Items {
				addPodImages(ns.Name, p, &result.BitnamiPods, &result.LegacyPods)
			}
		}
	}

	return result, nil
}

// GenerateImagesAuditMarkdown runs the images audit and returns a Markdown section.
func GenerateImagesAuditMarkdown(clientset *kubernetes.Clientset) (string, error) {
	result, err := AuditImages(clientset)
	if err != nil {
		return "", err
	}

	// Convert to [][]string for markdown table
	toRows := func(entries []ImageEntry, cols int) [][]string {
		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			if cols == 5 {
				rows = append(rows, []string{e.Namespace, e.Kind, e.Name, e.Container, e.Image})
			} else {
				rows = append(rows, []string{e.Namespace, e.Name, e.Container, e.Image})
			}
		}
		return rows
	}

	var sb strings.Builder
	sb.WriteString("## Images Audit (Bitnami vs BitnamiLegacy)\n\n")
	sb.WriteString("Esta seção apresenta os pods que utilizam imagens Bitnami como runtime dentro do cluster. É importante destacar que o repositório público da Bitnami foi descontinuado, o que reforça a necessidade de revisar essas dependências, planejar a migração para repositórios mantidos e assegurar a continuidade, segurança e padronização do ambiente Kubernetes.\n\n")

	sb.WriteString("### Summary\n\n")
	sb.WriteString(BuildMarkdownTable([]string{"Category", "Count"}, [][]string{
		{"Controllers using Bitnami", fmt.Sprintf("%d", len(result.BitnamiControllers))},
		{"Controllers using BitnamiLegacy", fmt.Sprintf("%d", len(result.LegacyControllers))},
		{"Pods using Bitnami", fmt.Sprintf("%d", len(result.BitnamiPods))},
		{"Pods using BitnamiLegacy", fmt.Sprintf("%d", len(result.LegacyPods))},
	}))
	sb.WriteString("\n")

	if len(result.BitnamiControllers) > 0 {
		sb.WriteString("### Controllers using Bitnami (migrate to BitnamiLegacy)\n\n")
		sb.WriteString(BuildMarkdownTable([]string{"Namespace", "Kind", "Name", "Container", "Image"}, toRows(result.BitnamiControllers, 5)))
		sb.WriteString("\n")
	}
	if len(result.LegacyControllers) > 0 {
		sb.WriteString("### Controllers already using BitnamiLegacy\n\n")
		sb.WriteString(BuildMarkdownTable([]string{"Namespace", "Kind", "Name", "Container", "Image"}, toRows(result.LegacyControllers, 5)))
		sb.WriteString("\n")
	}
	if len(result.BitnamiPods) > 0 {
		sb.WriteString("### Pods using Bitnami (runtime)\n\n")
		sb.WriteString(BuildMarkdownTable([]string{"Namespace", "Pod", "Container", "Image"}, toRows(result.BitnamiPods, 4)))
		sb.WriteString("\n")
	}
	if len(result.LegacyPods) > 0 {
		sb.WriteString("### Pods using BitnamiLegacy (runtime)\n\n")
		sb.WriteString(BuildMarkdownTable([]string{"Namespace", "Pod", "Container", "Image"}, toRows(result.LegacyPods, 4)))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// vendorFromImage returns image vendor markers ("bitnami", "bitnamilegacy") or empty string.
func vendorFromImage(image string) string {
	lower := strings.ToLower(image)
	if strings.Contains(lower, "bitnamilegacy") {
		return "bitnamilegacy"
	}
	parts := strings.Split(lower, "/")
	if len(parts) == 0 {
		return ""
	}
	registry := "docker.io"
	pathStart := 0
	if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost" {
		registry = parts[0]
		pathStart = 1
	}
	if len(parts[pathStart:]) < 2 {
		return ""
	}
	org := parts[pathStart]
	if registry == "docker.io" && org == "bitnami" {
		return "bitnami"
	}
	return ""
}

func addControllerImages(namespace, kind, name string, spec corev1.PodSpec, bitnamiCtrls *[]ImageEntry, legacyCtrls *[]ImageEntry) {
	for _, c := range spec.InitContainers {
		switch vendorFromImage(c.Image) {
		case "bitnami":
			*bitnamiCtrls = append(*bitnamiCtrls, ImageEntry{namespace, kind, name, c.Name, c.Image})
		case "bitnamilegacy":
			*legacyCtrls = append(*legacyCtrls, ImageEntry{namespace, kind, name, c.Name, c.Image})
		}
	}
	for _, c := range spec.Containers {
		switch vendorFromImage(c.Image) {
		case "bitnami":
			*bitnamiCtrls = append(*bitnamiCtrls, ImageEntry{namespace, kind, name, c.Name, c.Image})
		case "bitnamilegacy":
			*legacyCtrls = append(*legacyCtrls, ImageEntry{namespace, kind, name, c.Name, c.Image})
		}
	}
}

func addPodImages(namespace string, p corev1.Pod, bitnamiPods *[]ImageEntry, legacyPods *[]ImageEntry) {
	for _, c := range p.Spec.InitContainers {
		switch vendorFromImage(c.Image) {
		case "bitnami":
			*bitnamiPods = append(*bitnamiPods, ImageEntry{namespace, "", p.Name, c.Name, c.Image})
		case "bitnamilegacy":
			*legacyPods = append(*legacyPods, ImageEntry{namespace, "", p.Name, c.Name, c.Image})
		}
	}
	for _, c := range p.Spec.Containers {
		switch vendorFromImage(c.Image) {
		case "bitnami":
			*bitnamiPods = append(*bitnamiPods, ImageEntry{namespace, "", p.Name, c.Name, c.Image})
		case "bitnamilegacy":
			*legacyPods = append(*legacyPods, ImageEntry{namespace, "", p.Name, c.Name, c.Image})
		}
	}
}
