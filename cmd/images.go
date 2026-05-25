package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flysecurity/a3k/internal/k8s"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "images",
		Short: "Audit container images (Bitnami vs BitnamiLegacy)",
		RunE:  runImages,
	})
}

func runImages(cmd *cobra.Command, args []string) error {
	cs, err := app.Client.Clientset()
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}
	result, err := k8s.AuditImages(cs)
	if err != nil {
		return err
	}
	app.Renderer.Header("Images Audit (Bitnami vs BitnamiLegacy)")
	app.Renderer.Table([]string{"Category", "Count"}, [][]string{
		{"Controllers using Bitnami", fmt.Sprintf("%d", len(result.BitnamiControllers))},
		{"Controllers using BitnamiLegacy", fmt.Sprintf("%d", len(result.LegacyControllers))},
		{"Pods using Bitnami", fmt.Sprintf("%d", len(result.BitnamiPods))},
		{"Pods using BitnamiLegacy", fmt.Sprintf("%d", len(result.LegacyPods))},
	})
	if len(result.BitnamiControllers) > 0 {
		app.Renderer.Subheader("Controllers using Bitnami (migrate to BitnamiLegacy)")
		rows := make([][]string, 0, len(result.BitnamiControllers))
		for _, e := range result.BitnamiControllers {
			rows = append(rows, []string{e.Namespace, e.Kind, e.Name, e.Container, e.Image})
		}
		app.Renderer.Table([]string{"Namespace", "Kind", "Name", "Container", "Image"}, rows)
	}
	if len(result.LegacyControllers) > 0 {
		app.Renderer.Subheader("Controllers already using BitnamiLegacy")
		rows := make([][]string, 0, len(result.LegacyControllers))
		for _, e := range result.LegacyControllers {
			rows = append(rows, []string{e.Namespace, e.Kind, e.Name, e.Container, e.Image})
		}
		app.Renderer.Table([]string{"Namespace", "Kind", "Name", "Container", "Image"}, rows)
	}
	if len(result.BitnamiPods) > 0 {
		app.Renderer.Subheader("Pods using Bitnami (runtime)")
		rows := make([][]string, 0, len(result.BitnamiPods))
		for _, e := range result.BitnamiPods {
			rows = append(rows, []string{e.Namespace, e.Name, e.Container, e.Image})
		}
		app.Renderer.Table([]string{"Namespace", "Pod", "Container", "Image"}, rows)
	}
	if len(result.LegacyPods) > 0 {
		app.Renderer.Subheader("Pods using BitnamiLegacy (runtime)")
		rows := make([][]string, 0, len(result.LegacyPods))
		for _, e := range result.LegacyPods {
			rows = append(rows, []string{e.Namespace, e.Name, e.Container, e.Image})
		}
		app.Renderer.Table([]string{"Namespace", "Pod", "Container", "Image"}, rows)
	}
	return nil
}
