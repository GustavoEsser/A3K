package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flysecurity/a3k/internal/k8s"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "nodes",
		Short:   "Show node inventory and resource allocation",
		Example: "  a3k nodes\n  a3k nodes --output raw",
		RunE:    runNodes,
	})
}

func runNodes(cmd *cobra.Command, args []string) error {
	cs, err := app.Client.Clientset()
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	summary, err := k8s.GetNodes(cs)
	if err != nil {
		return err
	}

	app.Renderer.Header("Cluster Nodes")

	quickRows := make([][]string, 0, len(summary.Nodes))
	for _, n := range summary.Nodes {
		quickRows = append(quickRows, []string{
			n.Name, n.InternalIP, n.MachineType,
			n.CPUAllocatable, n.CPURequested,
			n.MemAllocatable, n.MemRequested,
			fmt.Sprintf("%d", n.PodCount), n.Ready,
		})
	}
	app.Renderer.Table([]string{"Node", "InternalIP", "Type", "CPU Alloc", "CPU Req", "Mem Alloc", "Mem Req", "Pods", "Ready"}, quickRows)

	app.Renderer.Subheader("Totals")
	app.Renderer.Table([]string{"Metric", "Value"}, [][]string{
		{"Total Nodes", fmt.Sprintf("%d", summary.TotalNodes)},
		{"Total CPU", summary.TotalCPU},
		{"CPU Requested", summary.TotalCPURequested},
		{"Total Memory", summary.TotalMemory},
		{"Memory Requested", summary.TotalMemoryRequested},
	})

	app.Renderer.Subheader("Machine Types")
	mtRows := make([][]string, 0, len(summary.MachineTypes))
	for _, mt := range summary.MachineTypes {
		mtRows = append(mtRows, []string{mt.Name, fmt.Sprintf("%d", mt.Count)})
	}
	app.Renderer.Table([]string{"Type", "Nodes"}, mtRows)

	return nil
}
