package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/flysecurity/a3k/internal/k8s"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "events",
		Short: "Summarize cluster events (warnings, reasons, objects)",
		RunE:  runEvents,
	})
}

func runEvents(cmd *cobra.Command, args []string) error {
	cs, err := app.Client.Clientset()
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}
	summary, err := k8s.AnalyzeClusterEvents(cs)
	if err != nil {
		return err
	}
	app.Renderer.Header("Cluster Events Overview")
	app.Renderer.Table([]string{"Type", "Count"}, [][]string{
		{"Total Events", fmt.Sprintf("%d", summary.Total)},
		{"Warnings", fmt.Sprintf("%d", summary.Warnings)},
		{"Normals", fmt.Sprintf("%d", summary.Normals)},
	})
	if len(summary.TopWarningReasons) > 0 {
		app.Renderer.Subheader("Top Warning Reasons")
		rows := make([][]string, 0, len(summary.TopWarningReasons))
		for _, r := range summary.TopWarningReasons {
			rows = append(rows, []string{r.Key, fmt.Sprintf("%d", r.Count)})
		}
		app.Renderer.Table([]string{"Reason", "Count"}, rows)
	}
	if len(summary.TopWarningObjects) > 0 {
		app.Renderer.Subheader("Top Affected Objects")
		rows := make([][]string, 0, len(summary.TopWarningObjects))
		for _, o := range summary.TopWarningObjects {
			rows = append(rows, []string{o.Key, fmt.Sprintf("%d", o.Count)})
		}
		app.Renderer.Table([]string{"Object", "Count"}, rows)
	}
	if len(summary.RecentWarnings) > 0 {
		app.Renderer.Subheader("Recent Warnings")
		rows := make([][]string, 0, len(summary.RecentWarnings))
		for _, e := range summary.RecentWarnings {
			msg := e.Message
			if len(msg) > 80 {
				msg = msg[:77] + "..."
			}
			rows = append(rows, []string{
				e.Time.Format(time.RFC3339),
				fmt.Sprintf("%s/%s %s", e.Namespace, e.Name, e.Kind),
				e.Reason,
				msg,
			})
		}
		app.Renderer.Table([]string{"Time", "Object", "Reason", "Message"}, rows)
	}
	return nil
}
