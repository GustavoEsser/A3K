package config

import (
	"os"
	"path/filepath"
)

// Config holds all runtime configuration resolved from flags, env vars, and config file.
type Config struct {
	Kubeconfig string
	Output     string // "table" | "json" | "yaml" | "raw"
	Namespace  string
	Verbose    bool
	NoColor    bool
	// Report-specific
	ClusterName string
	Author      string
	ReportPath  string
}

// DefaultReportPath returns the default path for generated reports.
func DefaultReportPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "a3k-reports")
}
