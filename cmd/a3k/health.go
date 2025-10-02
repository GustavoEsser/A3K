package main

import (
    "context"
    "fmt"
    "sort"

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