```
                                    
                                           d8888  .d8888b.  888    d8P  
                                          d88888 d88P  Y88b 888   d8P   
                                         d88P888      .d88P 888  d8P    
                                        d88P 888     8888"  888d88K     
                                       d88P  888      "Y8b. 8888888b    
                                      d88P   888 888    888 888  Y88b   
                                     d8888888888 Y88b  d88P 888   Y88b  
                                    d88P     888  "Y8888P"  888    Y88b
                                                                     
                                Assessment · Audit · Analyzer for Kubernetes

```

![Release](https://img.shields.io/badge/release-v1.0.0-blue)
![Go Version](https://img.shields.io/badge/go-1.23%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey)

**A3K** is a command-line tool for auditing, analyzing, and reporting on Kubernetes clusters. It gives platform engineers and SREs a fast, opinionated overview of workload health, security posture, resource configuration, node inventory, cluster events, and container images — all from a single binary.

---

## Features

- Workload inventory: Deployments, ReplicaSets, StatefulSets, DaemonSets, CronJobs, Pods
- Node analysis: CPU/memory allocation, machine types, topology
- Cluster events: warning aggregation, top reasons, top affected objects
- Health overview: node conditions, pod failures, CrashLoopBackOff, pending PVCs
- Security audit: privileged containers, missing security contexts, root execution
- Image audit: detect Bitnami vs BitnamiLegacy container images
- Markdown report: comprehensive, timestamped cluster report saved locally
- Multiple output formats: `table` (gum-styled), `raw`, `json`
- Viper config file + environment variable support

---

## Installation

### Using `go install`

```bash
go install github.com/flysecurity/a3k@latest
```

### From source

```bash
git clone https://github.com/flysecurity/a3k.git
cd a3k
make build
# Binary is at ./a3k
```

### Install to `$GOPATH/bin`

```bash
make install
```

### Config file (optional)

Copy `configs/config.yaml` to `~/.a3k.yaml` or place an `a3k.yaml` in your working directory:

```bash
cp configs/config.yaml ~/.a3k.yaml
```

---

## Quick Start

```bash
# Show help
a3k --help

# List workloads across all namespaces
a3k workloads

# Show node inventory
a3k nodes

# Summarize cluster events
a3k events

# Health overview
a3k health

# Audit images (Bitnami vs BitnamiLegacy)
a3k images

# Generate a full Markdown report
a3k report --cluster-name my-prod-cluster --author "Jane Doe"

# Use a specific kubeconfig
a3k nodes --kubeconfig ~/.kube/my-cluster.yaml

# Raw (no color) output
a3k workloads --output raw

# JSON output
a3k workloads --output json
```

---

## Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `workloads` | `wl` | Show workload counts (Deployments, StatefulSets, DaemonSets, CronJobs, Pods) |
| `nodes` | — | Show node inventory and resource allocation |
| `events` | — | Summarize cluster events (warnings, reasons, affected objects) |
| `health` | — | Show cluster and workload health overview |
| `images` | — | Audit container images (Bitnami vs BitnamiLegacy) |
| `report` | — | Generate a comprehensive Markdown cluster report |
| `version` | — | Print version information |

### `workloads`

```bash
a3k workloads
a3k wl --output json
```

Counts Deployments, ReplicaSets, StatefulSets, DaemonSets, CronJobs, and Running Pods across all namespaces.

### `nodes`

```bash
a3k nodes
a3k nodes --output raw
```

Lists all nodes with CPU/memory allocatable and requested, machine type, pod count, and readiness status. Includes totals and machine type distribution.

### `events`

```bash
a3k events
```

Aggregates cluster-wide events showing total/warning/normal counts, top warning reasons, top affected objects, and the 10 most recent warnings.

### `health`

```bash
a3k health
```

Node conditions (NotReady, MemoryPressure, DiskPressure, PIDPressure, Unschedulable), problematic Deployments/StatefulSets/DaemonSets, pod failure states, top restart counts, and pending PVCs.

### `images`

```bash
a3k images
```

Scans all namespaces for containers using `docker.io/bitnami/*` or `bitnamilegacy` images, covering Deployments, StatefulSets, DaemonSets, CronJobs, and running Pods.

### `report`

```bash
a3k report
a3k report --cluster-name prod-eks --author "Gustavo Esser"
a3k report --cluster-name prod-eks --output raw
```

Generates a comprehensive Markdown report saved to `~/a3k-reports/a3k-report-<timestamp>.md` with secure permissions (0600). The report includes all sections: cluster overview, workloads, nodes, ingresses, resources, health, security audit, events, and image audit.

---

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `~/.kube/config` | Path to kubeconfig file |
| `--output` | `table` | Output format: `table`, `raw`, `json` |
| `--namespace` | (all) | Target namespace (future use) |
| `--verbose` | `false` | Enable debug logging to stderr |
| `--no-color` | `false` | Disable gum-based styled output |
| `--config` | (auto) | Config file path |

---

## Config File Reference

Place at `~/.a3k.yaml` or `./a3k.yaml`:

```yaml
# A3K configuration file

# kubeconfig: ~/.kube/config
output: table
# namespace: ""
report_path: ~/a3k-reports
no_color: false
verbose: false
```

---

## Environment Variables

All flags can be overridden via environment variables with the `A3K_` prefix:

| Variable | Corresponding Flag | Description |
|----------|--------------------|-------------|
| `A3K_KUBECONFIG` | `--kubeconfig` | Path to kubeconfig file |
| `A3K_OUTPUT` | `--output` | Output format |
| `A3K_NAMESPACE` | `--namespace` | Target namespace |
| `A3K_VERBOSE` | `--verbose` | Enable verbose logging |
| `A3K_NO_COLOR` | `--no-color` | Disable colored output |
| `A3K_REPORT_PATH` | — | Directory for generated reports |

Example:

```bash
export A3K_KUBECONFIG=~/.kube/prod.yaml
export A3K_OUTPUT=raw
a3k nodes
```

---

## Project Structure

```
.
├── main.go                     # Entry point
├── cmd/                        # Cobra commands (package cmd)
│   ├── root.go                 # Root command + App context + Viper setup
│   ├── workloads.go
│   ├── nodes.go
│   ├── events.go
│   ├── health.go
│   ├── report.go
│   ├── images.go
│   └── version.go
├── internal/
│   ├── config/
│   │   └── config.go           # Config struct
│   ├── k8s/                    # Kubernetes business logic
│   │   ├── client.go           # Lazy clientset factory
│   │   ├── cluster.go          # Cluster info / provider detection
│   │   ├── workloads.go        # Workload counts + markdown
│   │   ├── nodes.go            # Node inventory + resource totals
│   │   ├── events.go           # Event aggregation + markdown
│   │   ├── health.go           # Health data + markdown
│   │   ├── images.go           # Image audit (Bitnami)
│   │   ├── ingresses.go        # Ingress listing + markdown
│   │   ├── resources.go        # Resource requests/limits audit
│   │   ├── security.go         # Security posture audit + markdown
│   │   └── markdown.go         # Shared BuildMarkdownTable / EscapeCell
│   ├── output/                 # Renderer interface + implementations
│   │   ├── renderer.go         # Interface + factory
│   │   ├── table.go            # gum-backed TableRenderer
│   │   └── json.go             # JSONRenderer + RawRenderer
│   ├── report/                 # Report generation
│   │   ├── markdown.go         # Generate() full report string
│   │   └── writer.go           # Secure file writer
│   └── logging/
│       └── logger.go           # slog-based structured logger
├── configs/
│   └── config.yaml             # Example/default configuration
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Roadmap

- [ ] HTML report export
- [ ] JSON/YAML structured report output
- [ ] RBAC audit (ServiceAccount permissions, ClusterRoleBindings)
- [ ] CVE / vulnerability scanning integration
- [ ] Cost analysis (resource utilization vs. requests)
- [ ] AI-assisted analysis and recommendations
- [ ] TUI interactive mode (bubbletea)
- [ ] GitHub Actions integration (report as PR comment)
- [ ] Namespace-scoped analysis (`--namespace` filtering)
- [ ] Diff reports (compare two clusters or two points in time)

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes and add tests
4. Run `make test` and `make lint`
5. Open a pull request

---

## License

MIT — see [LICENSE](LICENSE) for details.
