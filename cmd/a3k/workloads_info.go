package main

import (
    "context"
    "fmt"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

// GenerateWorkloadsInfoMarkdown returns a Markdown section summarizing workload counts across the cluster.
// It counts Deployments, ReplicaSets, StatefulSets, CronJobs, and Pods across all namespaces.
func GenerateWorkloadsInfoMarkdown(clientset *kubernetes.Clientset) (string, error) {
    ctx := context.TODO()

    nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
    if err != nil {
        return "", fmt.Errorf("error getting namespaces: %v", err)
    }

    var deployments, replicasets, statefulsets, cronjobs, pods int

    for _, ns := range nsList.Items {
        if deps, err := clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            deployments += len(deps.Items)
        }
        if rss, err := clientset.AppsV1().ReplicaSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            replicasets += len(rss.Items)
        }
        if sss, err := clientset.AppsV1().StatefulSets(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            statefulsets += len(sss.Items)
        }
        if cjs, err := clientset.BatchV1().CronJobs(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            cronjobs += len(cjs.Items)
        }
        if pds, err := clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{}); err == nil {
            pods += len(pds.Items)
        }
    }

    rows := [][]string{
        {"Deployment", fmt.Sprintf("%d", deployments)},
        {"ReplicaSet", fmt.Sprintf("%d", replicasets)},
        {"StatefulSet", fmt.Sprintf("%d", statefulsets)},
        {"CronJob", fmt.Sprintf("%d", cronjobs)},
        {"Pod", fmt.Sprintf("%d", pods)},
    }

    md := "## Resumo de Workloads 📊\n\n"
    md += "Esta seção apresenta um panorama dos workloads em execução no cluster, abrangendo diferentes tipos de recursos responsáveis por gerenciar aplicações e tarefas no Kubernetes.\n\n"
    md += buildMarkdownTable([]string{"Tipo", "Quantidade"}, rows)
    md += "\n"
    return md, nil
}
