package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/flysecurity/a3k/internal/k8s"
	"github.com/flysecurity/a3k/internal/report"
)

func init() {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate a comprehensive Markdown cluster report",
		Example: `  a3k report
  a3k report --cluster-name prod-eks --author "Gustavo Esser"`,
		RunE: runReport,
	}
	cmd.Flags().String("cluster-name", "", "cluster name for the report header")
	cmd.Flags().String("author", "", "author name for the report header")
	_ = viper.BindPFlag("cluster_name", cmd.Flags().Lookup("cluster-name"))
	_ = viper.BindPFlag("author", cmd.Flags().Lookup("author"))
	rootCmd.AddCommand(cmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	cs, err := app.Client.Clientset()
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	app.Logger.Info("collecting cluster data...")

	clusterInfo, err := k8s.GetClusterInfo(cs)
	if err != nil {
		return fmt.Errorf("cluster info: %w", err)
	}

	nodes, err := k8s.GetNodes(cs)
	if err != nil {
		return fmt.Errorf("nodes: %w", err)
	}

	workloadsMD, err := k8s.GenerateWorkloadsInfoMarkdown(cs)
	if err != nil {
		return fmt.Errorf("workloads: %w", err)
	}

	ingressMD, err := k8s.GenerateIngressesMarkdown(cs)
	if err != nil {
		return fmt.Errorf("ingresses: %w", err)
	}

	resources, err := k8s.AnalyzeWorkloadResources(cs)
	if err != nil {
		return fmt.Errorf("resources: %w", err)
	}

	healthMD, err := k8s.GenerateHealthMarkdown(cs)
	if err != nil {
		return fmt.Errorf("health: %w", err)
	}

	securityMD, err := k8s.GenerateSecurityAuditMarkdown(cs)
	if err != nil {
		return fmt.Errorf("security: %w", err)
	}

	eventsSummary, err := k8s.AnalyzeClusterEvents(cs)
	if err != nil {
		return fmt.Errorf("events: %w", err)
	}

	imagesMD, err := k8s.GenerateImagesAuditMarkdown(cs)
	if err != nil {
		return fmt.Errorf("images: %w", err)
	}

	content := report.Generate(
		clusterInfo, workloadsMD, nodes, ingressMD, resources,
		healthMD, securityMD, eventsSummary, imagesMD,
		report.Options{
			ClusterName: viper.GetString("cluster_name"),
			Author:      viper.GetString("author"),
		},
	)

	w := report.NewWriter(app.Config.ReportPath)
	path, err := w.Save(content)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Report generated: %s\n", path)
	return nil
}
