package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// WorkloadSummary holds cluster-wide workload counts.
type WorkloadSummary struct {
	Deployments  int
	ReplicaSets  int
	StatefulSets int
	DaemonSets   int
	CronJobs     int
	RunningPods  int
	TotalPods    int
}

// GetWorkloadSummary counts workloads cluster-wide using the kubernetes client.
func GetWorkloadSummary(clientset *kubernetes.Clientset) (*WorkloadSummary, error) {
	ctx := context.Background()

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting namespaces: %w", err)
	}

	summary := &WorkloadSummary{}

	for _, ns := range namespaces.Items {
		namespace := ns.Name

		if deps, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{}); err == nil {
			summary.Deployments += len(deps.Items)
		}

		if rss, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{}); err == nil {
			summary.ReplicaSets += len(rss.Items)
		}

		if sss, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{}); err == nil {
			summary.StatefulSets += len(sss.Items)
		}

		if dss, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{}); err == nil {
			summary.DaemonSets += len(dss.Items)
		}

		if cjs, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{}); err == nil {
			summary.CronJobs += len(cjs.Items)
		}

		if running, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{FieldSelector: "status.phase=Running"}); err == nil {
			summary.RunningPods += len(running.Items)
		}

		if all, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{}); err == nil {
			summary.TotalPods += len(all.Items)
		}
	}

	return summary, nil
}

// GenerateWorkloadsInfoMarkdown returns a Markdown section summarizing workload counts across the cluster.
func GenerateWorkloadsInfoMarkdown(clientset *kubernetes.Clientset) (string, error) {
	ctx := context.Background()

	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting namespaces: %w", err)
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
	md += BuildMarkdownTable([]string{"Tipo", "Quantidade"}, rows)
	md += "\n"
	return md, nil
}
