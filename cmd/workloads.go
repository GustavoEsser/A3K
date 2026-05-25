package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flysecurity/a3k/internal/k8s"
)

func init() {
	cmd := &cobra.Command{
		Use:     "workloads",
		Aliases: []string{"wl"},
		Short:   "Show workload counts (Deployments, StatefulSets, DaemonSets, CronJobs, Pods)",
		Example: "  a3k workloads\n  a3k wl --output json",
		RunE:    runWorkloads,
	}
	rootCmd.AddCommand(cmd)
}

func runWorkloads(cmd *cobra.Command, args []string) error {
	cs, err := app.Client.Clientset()
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	summary, err := k8s.GetWorkloadSummary(cs)
	if err != nil {
		return err
	}

	app.Renderer.Header("Workloads Overview")
	app.Renderer.Table(
		[]string{"Resource", "Count"},
		[][]string{
			{"Deployments", fmt.Sprintf("%d", summary.Deployments)},
			{"ReplicaSets", fmt.Sprintf("%d", summary.ReplicaSets)},
			{"StatefulSets", fmt.Sprintf("%d", summary.StatefulSets)},
			{"DaemonSets", fmt.Sprintf("%d", summary.DaemonSets)},
			{"CronJobs", fmt.Sprintf("%d", summary.CronJobs)},
			{"Running Pods", fmt.Sprintf("%d", summary.RunningPods)},
		},
	)
	return nil
}
