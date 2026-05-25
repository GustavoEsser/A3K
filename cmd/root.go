package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/flysecurity/a3k/internal/config"
	"github.com/flysecurity/a3k/internal/k8s"
	"github.com/flysecurity/a3k/internal/logging"
	"github.com/flysecurity/a3k/internal/output"
)

// Build-time variables injected via ldflags by GoReleaser.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// App holds shared dependencies injected into all subcommands.
type App struct {
	Config   *config.Config
	Client   *k8s.Client
	Logger   *slog.Logger
	Renderer output.Renderer
}

var (
	app     = &App{}
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "a3k",
		Short: "A3K — Assessment · Audit · Analyzer for Kubernetes",
		Long: `A3K is a CLI tool for auditing, analyzing and reporting on Kubernetes clusters.

It provides workload analysis, security posture assessment, node inventory,
event aggregation, and comprehensive markdown reports.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
)

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	pf := rootCmd.PersistentFlags()
	pf.StringVar(&cfgFile, "config", "", "config file (default: $HOME/.a3k.yaml or ./a3k.yaml)")
	pf.String("kubeconfig", "", "path to kubeconfig (default: $HOME/.kube/config)")
	pf.String("output", "table", "output format: table|raw|json")
	pf.String("namespace", "", "target namespace (default: all namespaces)")
	pf.Bool("verbose", false, "enable verbose/debug logging")
	pf.Bool("no-color", false, "disable colored output")

	_ = viper.BindPFlag("kubeconfig", pf.Lookup("kubeconfig"))
	_ = viper.BindPFlag("output", pf.Lookup("output"))
	_ = viper.BindPFlag("namespace", pf.Lookup("namespace"))
	_ = viper.BindPFlag("verbose", pf.Lookup("verbose"))
	_ = viper.BindPFlag("no_color", pf.Lookup("no-color"))

	viper.SetEnvPrefix("A3K")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("output", "table")
	viper.SetDefault("namespace", "")
	viper.SetDefault("report_path", "")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cfg := &config.Config{
			Kubeconfig: viper.GetString("kubeconfig"),
			Output:     viper.GetString("output"),
			Namespace:  viper.GetString("namespace"),
			Verbose:    viper.GetBool("verbose"),
			NoColor:    viper.GetBool("no_color"),
			ReportPath: viper.GetString("report_path"),
		}
		app.Config = cfg
		app.Client = k8s.NewClient(cfg.Kubeconfig)
		app.Logger = logging.New(cfg.Verbose)
		app.Renderer = output.New(cfg.Output, cfg.NoColor)
		return nil
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".a3k")
		viper.SetConfigType("yaml")
	}
	_ = viper.ReadInConfig()
}
