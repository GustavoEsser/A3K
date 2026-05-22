package main

import (
    "context"
    "fmt"
    "sort"
    "strings"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

// getHealth prints a consolidated health overview for the cluster
func getHealth(clientset *kubernetes.Clientset) error {
    printHeader("Cluster Health Overview")

    ctx := context.TODO()

    // Nodes health
    nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
    if err != nil {
        return fmt.Errorf("error getting nodes: %v", err)
    }

    var notReady, memPressure, diskPressure, pidPressure, unsched int
    var criticalNodeRows [][]string
    for _, n := range nodes.Items {
        ready := "Unknown"
        hasMem := false
        hasDisk := false
        hasPID := false
        for _, c := range n.Status.Conditions {
            switch c.Type {
            case corev1.NodeReady:
                if c.Status == corev1.ConditionTrue {
                    ready = "Ready"
                } else {
                    ready = string(c.Status)
                    notReady++
                }
            case corev1.NodeMemoryPressure:
                if c.Status == corev1.ConditionTrue { hasMem = true }
            case corev1.NodeDiskPressure:
                if c.Status == corev1.ConditionTrue { hasDisk = true }
            case corev1.NodePIDPressure:
                if c.Status == corev1.ConditionTrue { hasPID = true }
            }
        }
        if hasMem { memPressure++ }
        if hasDisk { diskPressure++ }
        if hasPID { pidPressure++ }
        if n.Spec.Unschedulable { unsched++ }

        if ready != "Ready" || hasMem || hasDisk || hasPID || n.Spec.Unschedulable {
            // Minimal info for quick triage
            criticalNodeRows = append(criticalNodeRows, []string{
                n.Name,
                ready,
                fmt.Sprintf("%t", hasMem),
                fmt.Sprintf("%t", hasDisk),
                fmt.Sprintf("%t", hasPID),
                fmt.Sprintf("%t", n.Spec.Unschedulable),
            })
        }
    }

    printSubheader("Nodes Summary")
    printTable([]string{"Metric", "Count"}, [][]string{
        {"Total Nodes", fmt.Sprintf("%d", len(nodes.Items))},
        {"Not Ready", fmt.Sprintf("%d", notReady)},
        {"Memory Pressure", fmt.Sprintf("%d", memPressure)},
        {"Disk Pressure", fmt.Sprintf("%d", diskPressure)},
        {"PID Pressure", fmt.Sprintf("%d", pidPressure)},
        {"Unschedulable", fmt.Sprintf("%d", unsched)},
    })
    if len(criticalNodeRows) > 0 {
        printSubheader("Critical Nodes")
        printTable([]string{"Node", "Ready", "MemPressure", "DiskPressure", "PIDPressure", "Unsched"}, criticalNodeRows)
    }

    // Workloads health across namespaces
    namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
    if err != nil {
        return fmt.Errorf("error getting namespaces: %v", err)
    }

    var depIssues, ssIssues, dsIssues int
    var podsPending, podsFailed, podsCrashLoop int
    type restartRow struct { ns, name string; restarts int }
    var restartRows []restartRow
    var depRows, ssRows, dsRows [][]string

    for _, ns := range namespaces.Items {
        // Deployments
        deps, err := clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{})
        if err == nil {
            for _, d := range deps.Items {
                desired := int32(0)
                if d.Spec.Replicas != nil { desired = *d.Spec.Replicas }
                available := d.Status.AvailableReplicas
                if available < desired {
                    depIssues++
                    depRows = append(depRows, []string{ns.Name, d.Name, fmt.Sprintf("%d", desired), fmt.Sprintf("%d", available)})
                }
            }
        }

        // StatefulSets
        sss, err := clientset.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{})
        if err == nil {
            for _, s := range sss.Items {
                replicas := int32(0)
                if s.Spec.Replicas != nil { replicas = *s.Spec.Replicas }
                ready := s.Status.ReadyReplicas
                if ready < replicas {
                    ssIssues++
                    ssRows = append(ssRows, []string{ns.Name, s.Name, fmt.Sprintf("%d", replicas), fmt.Sprintf("%d", ready)})
                }
            }
        }

        // DaemonSets
        dss, err := clientset.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{})
        if err == nil {
            for _, ds := range dss.Items {
                desired := ds.Status.DesiredNumberScheduled
                unavailable := ds.Status.NumberUnavailable
                if unavailable > 0 {
                    dsIssues++
                    dsRows = append(dsRows, []string{ns.Name, ds.Name, fmt.Sprintf("%d", desired), fmt.Sprintf("%d", unavailable)})
                }
            }
        }

        // Pods: phases, restarts, CrashLoop
        pods, err := clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
        if err == nil {
            for _, p := range pods.Items {
                switch p.Status.Phase {
                case corev1.PodPending:
                    podsPending++
                case corev1.PodFailed:
                    podsFailed++
                }
                // Restarts and CrashLoopBackOff
                var totalRestarts int
                var hasCrashLoop bool
                for _, cs := range p.Status.ContainerStatuses {
                    totalRestarts += int(cs.RestartCount)
                    if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
                        hasCrashLoop = true
                    }
                }
                if hasCrashLoop { podsCrashLoop++ }
                if totalRestarts >= 5 {
                    restartRows = append(restartRows, restartRow{ns: ns.Name, name: p.Name, restarts: totalRestarts})
                }
            }
        }
    }

    // Sort top restarts descending and limit to 10
    sort.Slice(restartRows, func(i, j int) bool { return restartRows[i].restarts > restartRows[j].restarts })
    if len(restartRows) > 10 { restartRows = restartRows[:10] }

    printSubheader("Workloads Summary")
    printTable([]string{"Metric", "Count"}, [][]string{
        {"Deployments not fully available", fmt.Sprintf("%d", depIssues)},
        {"StatefulSets not fully ready", fmt.Sprintf("%d", ssIssues)},
        {"DaemonSets unavailable", fmt.Sprintf("%d", dsIssues)},
        {"Pods Pending", fmt.Sprintf("%d", podsPending)},
        {"Pods Failed", fmt.Sprintf("%d", podsFailed)},
        {"Pods CrashLoopBackOff", fmt.Sprintf("%d", podsCrashLoop)},
    })

    if len(depRows) > 0 {
        printSubheader("Problematic Deployments")
        printTable([]string{"Namespace", "Deployment", "Desired", "Available"}, depRows)
    }
    if len(ssRows) > 0 {
        printSubheader("Problematic StatefulSets")
        printTable([]string{"Namespace", "StatefulSet", "Replicas", "Ready"}, ssRows)
    }
    if len(dsRows) > 0 {
        printSubheader("DaemonSets Unavailable")
        printTable([]string{"Namespace", "DaemonSet", "Desired", "Unavailable"}, dsRows)
    }

    if len(restartRows) > 0 {
        rows := make([][]string, 0, len(restartRows))
        for _, r := range restartRows {
            rows = append(rows, []string{r.ns, r.name, fmt.Sprintf("%d", r.restarts)})
        }
        printSubheader("Top Pods by Restarts (>=5)")
        printTable([]string{"Namespace", "Pod", "Restarts"}, rows)
    }

    // Services without endpoints
    var svcNoEpRows [][]string
    for _, ns := range namespaces.Items {
        eps, err := clientset.CoreV1().Endpoints(ns.Name).List(ctx, metav1.ListOptions{})
        if err == nil {
            for _, ep := range eps.Items {
                hasAddr := false
                for _, s := range ep.Subsets {
                    if len(s.Addresses) > 0 { hasAddr = true; break }
                }
                if !hasAddr {
                    svcNoEpRows = append(svcNoEpRows, []string{ns.Name, ep.Name})
                }
            }
        }
    }
    if len(svcNoEpRows) > 0 {
        printSubheader("Services Without Endpoints")
        printTable([]string{"Namespace", "Service"}, svcNoEpRows)
    }

    // PVCs pending
    var pvcPendingRows [][]string
    for _, ns := range namespaces.Items {
        pvcs, err := clientset.CoreV1().PersistentVolumeClaims(ns.Name).List(ctx, metav1.ListOptions{})
        if err == nil {
            for _, pvc := range pvcs.Items {
                if pvc.Status.Phase == corev1.ClaimPending {
                    req := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
                    sc := pvc.Spec.StorageClassName
                    scName := ""
                    if sc != nil { scName = *sc }
                    pvcPendingRows = append(pvcPendingRows, []string{ns.Name, pvc.Name, req.String(), scName})
                }
            }
        }
    }
    if len(pvcPendingRows) > 0 {
        printSubheader("Pending PVCs")
        printTable([]string{"Namespace", "PVC", "Requested", "StorageClass"}, pvcPendingRows)
    }

    // Events summary using existing analyzer
    summary, err := AnalyzeClusterEvents(clientset)
    if err == nil && summary != nil {
        printSubheader("Events Summary")
        printTable([]string{"Metric", "Count"}, [][]string{
            {"Total Events", fmt.Sprintf("%d", summary.Total)},
            {"Warnings", fmt.Sprintf("%d", summary.Warnings)},
            {"Normals", fmt.Sprintf("%d", summary.Normals)},
        })
    }

    return nil
}

// GenerateHealthMarkdown returns a Markdown section for the cluster health overview
func GenerateHealthMarkdown(clientset *kubernetes.Clientset) (string, error) {
    ctx := context.TODO()
    var sb strings.Builder
    sb.WriteString("## Saúde do Cluster 🏥\n\n")
    sb.WriteString("Esta seção apresenta um panorama da saúde dos nodes e workloads do cluster, incluindo condições críticas, pods com falhas e PVCs pendentes.\n\n")

    nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
    if err != nil {
        return "", fmt.Errorf("error getting nodes: %v", err)
    }

    var notReady, memPressure, diskPressure, pidPressure, unsched int
    var criticalNodeRows [][]string
    for _, n := range nodes.Items {
        ready := "Unknown"
        hasMem, hasDisk, hasPID := false, false, false
        for _, c := range n.Status.Conditions {
            switch c.Type {
            case corev1.NodeReady:
                if c.Status == corev1.ConditionTrue {
                    ready = "Ready"
                } else {
                    ready = string(c.Status)
                    notReady++
                }
            case corev1.NodeMemoryPressure:
                if c.Status == corev1.ConditionTrue { hasMem = true }
            case corev1.NodeDiskPressure:
                if c.Status == corev1.ConditionTrue { hasDisk = true }
            case corev1.NodePIDPressure:
                if c.Status == corev1.ConditionTrue { hasPID = true }
            }
        }
        if hasMem { memPressure++ }
        if hasDisk { diskPressure++ }
        if hasPID { pidPressure++ }
        if n.Spec.Unschedulable { unsched++ }
        if ready != "Ready" || hasMem || hasDisk || hasPID || n.Spec.Unschedulable {
            criticalNodeRows = append(criticalNodeRows, []string{
                n.Name, ready,
                fmt.Sprintf("%t", hasMem),
                fmt.Sprintf("%t", hasDisk),
                fmt.Sprintf("%t", hasPID),
                fmt.Sprintf("%t", n.Spec.Unschedulable),
            })
        }
    }

    sb.WriteString("### Resumo dos Nodes\n\n")
    sb.WriteString(buildMarkdownTable([]string{"Métrica", "Contagem"}, [][]string{
        {"Total de Nodes", fmt.Sprintf("%d", len(nodes.Items))},
        {"Not Ready", fmt.Sprintf("%d", notReady)},
        {"Memory Pressure", fmt.Sprintf("%d", memPressure)},
        {"Disk Pressure", fmt.Sprintf("%d", diskPressure)},
        {"PID Pressure", fmt.Sprintf("%d", pidPressure)},
        {"Unschedulable", fmt.Sprintf("%d", unsched)},
    }))
    sb.WriteString("\n")

    if len(criticalNodeRows) > 0 {
        sb.WriteString("### Nodes Críticos\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Node", "Ready", "MemPressure", "DiskPressure", "PIDPressure", "Unsched"}, criticalNodeRows))
        sb.WriteString("\n")
    }

    namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
    if err != nil {
        return "", fmt.Errorf("error getting namespaces: %v", err)
    }

    var podsPending, podsFailed, podsCrashLoop int
    var depRows, ssRows, dsRows [][]string
    type restartEntry struct {
        ns, name string
        restarts int
    }
    var restartRows []restartEntry

    for _, ns := range namespaces.Items {
        if deps, err := clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, d := range deps.Items {
                desired := int32(0)
                if d.Spec.Replicas != nil { desired = *d.Spec.Replicas }
                if d.Status.AvailableReplicas < desired {
                    depRows = append(depRows, []string{ns.Name, d.Name, fmt.Sprintf("%d", desired), fmt.Sprintf("%d", d.Status.AvailableReplicas)})
                }
            }
        }
        if sss, err := clientset.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, s := range sss.Items {
                replicas := int32(0)
                if s.Spec.Replicas != nil { replicas = *s.Spec.Replicas }
                if s.Status.ReadyReplicas < replicas {
                    ssRows = append(ssRows, []string{ns.Name, s.Name, fmt.Sprintf("%d", replicas), fmt.Sprintf("%d", s.Status.ReadyReplicas)})
                }
            }
        }
        if dss, err := clientset.AppsV1().DaemonSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, ds := range dss.Items {
                if ds.Status.NumberUnavailable > 0 {
                    dsRows = append(dsRows, []string{ns.Name, ds.Name, fmt.Sprintf("%d", ds.Status.DesiredNumberScheduled), fmt.Sprintf("%d", ds.Status.NumberUnavailable)})
                }
            }
        }
        if pods, err := clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, p := range pods.Items {
                switch p.Status.Phase {
                case corev1.PodPending:
                    podsPending++
                case corev1.PodFailed:
                    podsFailed++
                }
                var totalRestarts int
                var hasCrashLoop bool
                for _, cs := range p.Status.ContainerStatuses {
                    totalRestarts += int(cs.RestartCount)
                    if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
                        hasCrashLoop = true
                    }
                }
                if hasCrashLoop { podsCrashLoop++ }
                if totalRestarts >= 5 {
                    restartRows = append(restartRows, restartEntry{ns: ns.Name, name: p.Name, restarts: totalRestarts})
                }
            }
        }
    }

    sort.Slice(restartRows, func(i, j int) bool { return restartRows[i].restarts > restartRows[j].restarts })
    if len(restartRows) > 10 { restartRows = restartRows[:10] }

    sb.WriteString("### Resumo dos Workloads\n\n")
    sb.WriteString(buildMarkdownTable([]string{"Métrica", "Contagem"}, [][]string{
        {"Deployments indisponíveis", fmt.Sprintf("%d", len(depRows))},
        {"StatefulSets não prontos", fmt.Sprintf("%d", len(ssRows))},
        {"DaemonSets indisponíveis", fmt.Sprintf("%d", len(dsRows))},
        {"Pods Pending", fmt.Sprintf("%d", podsPending)},
        {"Pods Failed", fmt.Sprintf("%d", podsFailed)},
        {"Pods CrashLoopBackOff", fmt.Sprintf("%d", podsCrashLoop)},
    }))
    sb.WriteString("\n")

    if len(depRows) > 0 {
        sb.WriteString("### Deployments com Problemas\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Namespace", "Deployment", "Desejado", "Disponível"}, depRows))
        sb.WriteString("\n")
    }
    if len(ssRows) > 0 {
        sb.WriteString("### StatefulSets com Problemas\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Namespace", "StatefulSet", "Réplicas", "Prontos"}, ssRows))
        sb.WriteString("\n")
    }
    if len(dsRows) > 0 {
        sb.WriteString("### DaemonSets Indisponíveis\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Namespace", "DaemonSet", "Desejado", "Indisponível"}, dsRows))
        sb.WriteString("\n")
    }
    if len(restartRows) > 0 {
        rows := make([][]string, 0, len(restartRows))
        for _, r := range restartRows {
            rows = append(rows, []string{r.ns, r.name, fmt.Sprintf("%d", r.restarts)})
        }
        sb.WriteString("### Top Pods por Reinicializações (>=5)\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Namespace", "Pod", "Reinicializações"}, rows))
        sb.WriteString("\n")
    }

    var pvcRows [][]string
    for _, ns := range namespaces.Items {
        if pvcs, err := clientset.CoreV1().PersistentVolumeClaims(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            for _, pvc := range pvcs.Items {
                if pvc.Status.Phase == corev1.ClaimPending {
                    req := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
                    sc := ""
                    if pvc.Spec.StorageClassName != nil { sc = *pvc.Spec.StorageClassName }
                    pvcRows = append(pvcRows, []string{ns.Name, pvc.Name, req.String(), sc})
                }
            }
        }
    }
    if len(pvcRows) > 0 {
        sb.WriteString("### PVCs Pendentes\n\n")
        sb.WriteString(buildMarkdownTable([]string{"Namespace", "PVC", "Solicitado", "StorageClass"}, pvcRows))
        sb.WriteString("\n")
    }

    return sb.String(), nil
}