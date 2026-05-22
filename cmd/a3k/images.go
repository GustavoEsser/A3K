package main

import (
    "context"
    "fmt"
    "strings"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

// vendorFromImage returns image vendor markers ("bitnami", "bitnamilegacy") or empty string
func vendorFromImage(image string) string {
    lower := strings.ToLower(image)
    // Detect legacy explicitly anywhere in the ref
    if strings.Contains(lower, "bitnamilegacy") {
        return "bitnamilegacy"
    }

    // Only consider Bitnami if the image is from docker.io/bitnami/*
    // Parse registry and path: [registry]/[org]/[repo]:tag@digest
    parts := strings.Split(lower, "/")
    if len(parts) == 0 {
        return ""
    }

    // Determine registry: if first part has a '.' or ':' or equals 'localhost', it's an explicit registry.
    registry := "docker.io"
    pathStart := 0
    if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost" {
        registry = parts[0]
        pathStart = 1
    }

    // We need at least org/repo after registry (implicit or explicit)
    if len(parts[pathStart:]) < 2 {
        return ""
    }
    org := parts[pathStart]

    if registry == "docker.io" && org == "bitnami" {
        return "bitnami"
    }
    return ""
}

// GenerateImagesAuditMarkdown runs the images audit and returns a Markdown section
// that can be embedded into the cluster report.
func GenerateImagesAuditMarkdown(clientset *kubernetes.Clientset) (string, error) {
    ctx := context.TODO()
    namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
    if err != nil {
        return "", fmt.Errorf("error getting namespaces: %v", err)
    }

    var bitnamiControllers [][]string
    var legacyControllers [][]string
    var bitnamiPods [][]string
    var legacyPods [][]string

    for _, ns := range namespaces.Items {
        // Deployments
        if deps, err := clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, d := range deps.Items {
                addControllerImages(ns.Name, "Deployment", d.Name, d.Spec.Template.Spec, &bitnamiControllers, &legacyControllers)
            }
        }
        // StatefulSets
        if sss, err := clientset.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, s := range sss.Items {
                addControllerImages(ns.Name, "StatefulSet", s.Name, s.Spec.Template.Spec, &bitnamiControllers, &legacyControllers)
            }
        }
        // DaemonSets
        if dss, err := clientset.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, ds := range dss.Items {
                addControllerImages(ns.Name, "DaemonSet", ds.Name, ds.Spec.Template.Spec, &bitnamiControllers, &legacyControllers)
            }
        }
        // CronJobs
        if cjs, err := clientset.BatchV1().CronJobs(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, cj := range cjs.Items {
                addControllerImages(ns.Name, "CronJob", cj.Name, cj.Spec.JobTemplate.Spec.Template.Spec, &bitnamiControllers, &legacyControllers)
            }
        }

        // Pods (runtime validation)
        if pods, err := clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, p := range pods.Items {
                addPodImages(ns.Name, p, &bitnamiPods, &legacyPods)
            }
        }
    }

    var sb strings.Builder
    sb.WriteString("## Images Audit (Bitnami vs BitnamiLegacy)\n\n")
    sb.WriteString("Esta seção apresenta os pods que utilizam imagens Bitnami como runtime dentro do cluster. É importante destacar que o repositório público da Bitnami foi descontinuado, o que reforça a necessidade de revisar essas dependências, planejar a migração para repositórios mantidos e assegurar a continuidade, segurança e padronização do ambiente Kubernetes.\n\n")
    // Summary
    sb.WriteString("### Summary\n\n")
    sb.WriteString(buildMarkdownTable([]string{"Category", "Count"}, [][]string{
        {"Controllers using Bitnami", fmt.Sprintf("%d", len(bitnamiControllers))},
        {"Controllers using BitnamiLegacy", fmt.Sprintf("%d", len(legacyControllers))},
        {"Pods using Bitnami", fmt.Sprintf("%d", len(bitnamiPods))},
        {"Pods using BitnamiLegacy", fmt.Sprintf("%d", len(legacyPods))},
    }))
    sb.WriteString("\n")

    if len(bitnamiControllers) > 0 {
        sb.WriteString("### Controllers using Bitnami (migrate to BitnamiLegacy)\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Namespace", "Kind", "Name", "Container", "Image"}, bitnamiControllers))
        sb.WriteString("\n")
    }
    if len(legacyControllers) > 0 {
        sb.WriteString("### Controllers already using BitnamiLegacy\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Namespace", "Kind", "Name", "Container", "Image"}, legacyControllers))
        sb.WriteString("\n")
    }
    if len(bitnamiPods) > 0 {
        sb.WriteString("### Pods using Bitnami (runtime)\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Namespace", "Pod", "Container", "Image"}, bitnamiPods))
        sb.WriteString("\n")
    }
    if len(legacyPods) > 0 {
        sb.WriteString("### Pods using BitnamiLegacy (runtime)\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Namespace", "Pod", "Container", "Image"}, legacyPods))
        sb.WriteString("\n")
    }

    return sb.String(), nil
}

// escapeCell sanitizes a value for safe embedding in a Markdown table cell.
func escapeCell(s string) string {
    s = strings.ReplaceAll(s, "|", "\\|")
    s = strings.ReplaceAll(s, "\n", " ")
    s = strings.ReplaceAll(s, "\r", "")
    return s
}

// buildMarkdownTable renders a GitHub-flavored markdown table, escaping cell values.
func buildMarkdownTable(headers []string, rows [][]string) string {
    var sb strings.Builder
    // Header
    sb.WriteString("| ")
    sb.WriteString(strings.Join(headers, " | "))
    sb.WriteString(" |\n")
    // Separator
    sb.WriteString("| ")
    for i := range headers {
        if i > 0 {
            sb.WriteString(" | ")
        }
        sb.WriteString("---")
    }
    sb.WriteString(" |\n")
    // Rows — escape each cell to prevent broken table structure
    for _, r := range rows {
        escaped := make([]string, len(r))
        for i, cell := range r {
            escaped[i] = escapeCell(cell)
        }
        sb.WriteString("| ")
        sb.WriteString(strings.Join(escaped, " | "))
        sb.WriteString(" |\n")
    }
    return sb.String()
}

// auditImages aggregates controllers and pods using Bitnami/BitnamiLegacy images
func auditImages(clientset *kubernetes.Clientset) error {
    printHeader("Images Audit (Bitnami vs BitnamiLegacy)")

    ctx := context.TODO()
    namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
    if err != nil {
        return fmt.Errorf("error getting namespaces: %v", err)
    }

    var bitnamiControllers [][]string
    var legacyControllers [][]string
    var bitnamiPods [][]string
    var legacyPods [][]string

    for _, ns := range namespaces.Items {
        // Deployments
        if deps, err := clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, d := range deps.Items {
                addControllerImages(ns.Name, "Deployment", d.Name, d.Spec.Template.Spec, &bitnamiControllers, &legacyControllers)
            }
        }
        // StatefulSets
        if sss, err := clientset.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, s := range sss.Items {
                addControllerImages(ns.Name, "StatefulSet", s.Name, s.Spec.Template.Spec, &bitnamiControllers, &legacyControllers)
            }
        }
        // DaemonSets
        if dss, err := clientset.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, ds := range dss.Items {
                addControllerImages(ns.Name, "DaemonSet", ds.Name, ds.Spec.Template.Spec, &bitnamiControllers, &legacyControllers)
            }
        }
        // CronJobs
        if cjs, err := clientset.BatchV1().CronJobs(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, cj := range cjs.Items {
                addControllerImages(ns.Name, "CronJob", cj.Name, cj.Spec.JobTemplate.Spec.Template.Spec, &bitnamiControllers, &legacyControllers)
            }
        }

        // Pods (runtime validation)
        if pods, err := clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, p := range pods.Items {
                addPodImages(ns.Name, p, &bitnamiPods, &legacyPods)
            }
        }
    }

    // Summary
    printSubheader("Summary")
    printTable([]string{"Category", "Count"}, [][]string{
        {"Controllers using Bitnami", fmt.Sprintf("%d", len(bitnamiControllers))},
        {"Controllers using BitnamiLegacy", fmt.Sprintf("%d", len(legacyControllers))},
        {"Pods using Bitnami", fmt.Sprintf("%d", len(bitnamiPods))},
        {"Pods using BitnamiLegacy", fmt.Sprintf("%d", len(legacyPods))},
    })

    if len(bitnamiControllers) > 0 {
        printSubheader("Controllers using Bitnami (migrate to BitnamiLegacy)")
        printTable([]string{"Namespace", "Kind", "Name", "Container", "Image"}, bitnamiControllers)
    }
    if len(legacyControllers) > 0 {
        printSubheader("Controllers already using BitnamiLegacy")
        printTable([]string{"Namespace", "Kind", "Name", "Container", "Image"}, legacyControllers)
    }
    if len(bitnamiPods) > 0 {
        printSubheader("Pods using Bitnami (runtime)")
        printTable([]string{"Namespace", "Pod", "Container", "Image"}, bitnamiPods)
    }
    if len(legacyPods) > 0 {
        printSubheader("Pods using BitnamiLegacy (runtime)")
        printTable([]string{"Namespace", "Pod", "Container", "Image"}, legacyPods)
    }

    return nil
}

func addControllerImages(namespace, kind, name string, spec corev1.PodSpec, bitnamiCtrls *[][]string, legacyCtrls *[][]string) {
    for _, c := range spec.InitContainers {
        v := vendorFromImage(c.Image)
        if v == "bitnami" {
            *bitnamiCtrls = append(*bitnamiCtrls, []string{namespace, kind, name, c.Name, c.Image})
        } else if v == "bitnamilegacy" {
            *legacyCtrls = append(*legacyCtrls, []string{namespace, kind, name, c.Name, c.Image})
        }
    }
    for _, c := range spec.Containers {
        v := vendorFromImage(c.Image)
        if v == "bitnami" {
            *bitnamiCtrls = append(*bitnamiCtrls, []string{namespace, kind, name, c.Name, c.Image})
        } else if v == "bitnamilegacy" {
            *legacyCtrls = append(*legacyCtrls, []string{namespace, kind, name, c.Name, c.Image})
        }
    }
}

func addPodImages(namespace string, p corev1.Pod, bitnamiPods *[][]string, legacyPods *[][]string) {
    for _, c := range p.Spec.InitContainers {
        v := vendorFromImage(c.Image)
        if v == "bitnami" {
            *bitnamiPods = append(*bitnamiPods, []string{namespace, p.Name, c.Name, c.Image})
        } else if v == "bitnamilegacy" {
            *legacyPods = append(*legacyPods, []string{namespace, p.Name, c.Name, c.Image})
        }
    }
    for _, c := range p.Spec.Containers {
        v := vendorFromImage(c.Image)
        if v == "bitnami" {
            *bitnamiPods = append(*bitnamiPods, []string{namespace, p.Name, c.Name, c.Image})
        } else if v == "bitnamilegacy" {
            *legacyPods = append(*legacyPods, []string{namespace, p.Name, c.Name, c.Image})
        }
    }
}