# A3K - Assessment · Audit · Analyzer for Kubernetes

A3K is a command-line tool for analyzing and auditing Kubernetes clusters. It provides comprehensive insights into your cluster's configuration, resources, and potential issues.

## Features

- **Cluster Information**: Get details about your Kubernetes cluster including provider, version, and uptime
- **Resource Analysis**: Identify workloads missing resource requests or limits
- **Node Details**: View detailed information about cluster nodes including resources and versions
- **Comprehensive Reporting**: Generate detailed markdown reports with a single command

## Installation

### Prerequisites

- Go 1.16 or later
- Access to a Kubernetes cluster (via kubeconfig)

### Build from source

```bash
git clone https://github.com/GustavoEsser/a3k.git
cd a3k
go build -o a3k ./cmd/a3k
sudo mv a3k /usr/local/bin/
```

## Usage

### Generate a Cluster Report

Generate a comprehensive report about your Kubernetes cluster:

```bash
# Using default kubeconfig (~/.kube/config)
./a3k report

# Or specify a custom kubeconfig
./a3k --kubeconfig /path/to/kubeconfig report
```

The report will be saved to `~/a3k-reports/` with a timestamp in the filename.

### Report Contents

The generated report includes:

- Cluster information (provider, version, uptime)
- Resource analysis (workloads missing requests/limits)
- Node details and resource allocation
- Summary of cluster resources

## Examples

```bash
# Generate a report
./a3k report

# View the generated report
cat ~/a3k-reports/a3k-report-*.md
```

## Command Reference

```text
A3K - Assessment · Audit · Analyzer for Kubernetes

Usage:
  a3k [command] [flags]

Available Commands:
  workloads   Show information about workloads (Deployments, StatefulSets, DaemonSets, CronJobs, Pods)
  nodes       Show information about nodes (count, CPU, Memory, machine types)
  events      Show cluster events summary (warnings, reasons, objects)
  health      Show consolidated cluster and workloads health overview
  report      Generate a comprehensive markdown report of your Kubernetes cluster.
  images      Audit images (Bitnami vs BitnamiLegacy)
  help        Show this help message

Flags:
  --kubeconfig string   Path to kubeconfig file (default is $KUBECONFIG or $HOME/.kube/config)
  --help                Show help message
  -o raw                Output mode; 'raw' disables styling
  --output raw          Same as -o
```

### Examples

Show information about workloads:

```bash
a3k workloads
```

Show information about nodes:

```bash
a3k nodes
```

Show cluster events summary:

```bash
a3k events
```

Show consolidated cluster health overview:

```bash
a3k health
```

Generate a comprehensive markdown report:

```bash
a3k report
```

Specify a custom kubeconfig file:

```bash
a3k --kubeconfig /path/to/kubeconfig workloads
```

Force plain, non-interactive output:

```bash
a3k -o raw health
```

Audit images for Bitnami usage (and BitnamiLegacy):

```bash
a3k images
# Plain output (useful for CI/scripts)
a3k -o raw images
```

## License

MIT
