package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flysecurity/a3k/internal/k8s"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "health",
		Short: "Show cluster and workload health overview",
		RunE:  runHealth,
	})
}

func runHealth(cmd *cobra.Command, args []string) error {
	cs, err := app.Client.Clientset()
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}
	data, err := k8s.GetHealthData(cs)
	if err != nil {
		return err
	}
	app.Renderer.Header("Cluster Health Overview")
	app.Renderer.Subheader("Nodes Summary")
	app.Renderer.Table([]string{"Metric", "Count"}, [][]string{
		{"Total Nodes", fmt.Sprintf("%d", data.TotalNodes)},
		{"Not Ready", fmt.Sprintf("%d", data.NotReady)},
		{"Memory Pressure", fmt.Sprintf("%d", data.MemPressure)},
		{"Disk Pressure", fmt.Sprintf("%d", data.DiskPressure)},
		{"PID Pressure", fmt.Sprintf("%d", data.PIDPressure)},
		{"Unschedulable", fmt.Sprintf("%d", data.Unschedulable)},
	})
	if len(data.CriticalNodes) > 0 {
		app.Renderer.Subheader("Critical Nodes")
		app.Renderer.Table([]string{"Node", "Ready", "MemPressure", "DiskPressure", "PIDPressure", "Unsched"}, data.CriticalNodes)
	}
	app.Renderer.Subheader("Workloads Summary")
	app.Renderer.Table([]string{"Metric", "Count"}, [][]string{
		{"Deployments not fully available", fmt.Sprintf("%d", len(data.ProblematicDeployments))},
		{"StatefulSets not fully ready", fmt.Sprintf("%d", len(data.ProblematicStatefulSets))},
		{"DaemonSets unavailable", fmt.Sprintf("%d", len(data.ProblematicDaemonSets))},
		{"Pods Pending", fmt.Sprintf("%d", data.PodsPending)},
		{"Pods Failed", fmt.Sprintf("%d", data.PodsFailed)},
		{"Pods CrashLoopBackOff", fmt.Sprintf("%d", data.PodsCrashLoop)},
	})
	if len(data.ProblematicDeployments) > 0 {
		app.Renderer.Subheader("Problematic Deployments")
		app.Renderer.Table([]string{"Namespace", "Deployment", "Desired", "Available"}, data.ProblematicDeployments)
	}
	if len(data.ProblematicStatefulSets) > 0 {
		app.Renderer.Subheader("Problematic StatefulSets")
		app.Renderer.Table([]string{"Namespace", "StatefulSet", "Replicas", "Ready"}, data.ProblematicStatefulSets)
	}
	if len(data.ProblematicDaemonSets) > 0 {
		app.Renderer.Subheader("DaemonSets Unavailable")
		app.Renderer.Table([]string{"Namespace", "DaemonSet", "Desired", "Unavailable"}, data.ProblematicDaemonSets)
	}
	if len(data.TopRestarts) > 0 {
		app.Renderer.Subheader("Top Pods by Restarts (>=5)")
		app.Renderer.Table([]string{"Namespace", "Pod", "Restarts"}, data.TopRestarts)
	}
	if len(data.PendingPVCs) > 0 {
		app.Renderer.Subheader("Pending PVCs")
		app.Renderer.Table([]string{"Namespace", "PVC", "Requested", "StorageClass"}, data.PendingPVCs)
	}
	return nil
}
