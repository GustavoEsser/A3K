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
    // Check legacy first to avoid matching the substring "bitnami" inside bitnamilegacy
    if strings.Contains(lower, "bitnamilegacy") {
        return "bitnamilegacy"
    }
    // Detect any segment named bitnami (with or without registry prefix)
    parts := strings.Split(lower, "/")
    for _, p := range parts {
        if p == "bitnami" || strings.HasPrefix(p, "bitnami-") || strings.Contains(p, "bitnami") {
            return "bitnami"
        }
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

// buildMarkdownTable renders a simple GitHub-flavored markdown table.
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
    // Rows
    for _, r := range rows {
        sb.WriteString("| ")
        sb.WriteString(strings.Join(r, " | "))
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