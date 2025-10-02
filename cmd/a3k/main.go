package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func main() {
    // Pre-scan args to allow flags after subcommand (e.g., `a3k report --cluster-name X`)
    // Standard flag parsing stops at first non-flag; we support both forms: --key value and --key=value
    for i := 1; i < len(os.Args); i++ {
        arg := os.Args[i]
        if arg == "--cluster-name" {
            if i+1 < len(os.Args) {
                os.Setenv("A3K_CLUSTER_NAME", os.Args[i+1])
                i++
            }
            continue
        }
        if strings.HasPrefix(arg, "--cluster-name=") {
            os.Setenv("A3K_CLUSTER_NAME", strings.TrimPrefix(arg, "--cluster-name="))
            continue
        }
        if arg == "--author" {
            if i+1 < len(os.Args) {
                os.Setenv("A3K_AUTHOR", os.Args[i+1])
                i++
            }
            continue
        }
        if strings.HasPrefix(arg, "--author=") {
            os.Setenv("A3K_AUTHOR", strings.TrimPrefix(arg, "--author="))
            continue
        }
    }

    // Initialize command line flags
    kubeconfig := flag.String("kubeconfig", "", "path to the kubeconfig file (default is $HOME/.kube/config)")
    help := flag.Bool("help", false, "show help message")
    // Output mode: -o raw or --output raw disables gum styling
    output := flag.String("o", "", "output mode (use 'raw' to disable styling)")
    flag.StringVar(output, "output", "", "same as -o")
    // Report metadata
    clusterName := flag.String("cluster-name", "", "cluster name to display in the report header")
    author := flag.String("author", "", "author name to display in the report header")
    flag.Parse()

    // Apply output mode
    if *output == "raw" {
        // Disable gum styling globally
        os.Setenv("A3K_NO_GUM", "1")
    }

    if *help {
        showHelp()
        os.Exit(0)
    }

    // Export optional report metadata to environment variables for downstream usage
    if *clusterName != "" {
        os.Setenv("A3K_CLUSTER_NAME", *clusterName)
    }
    if *author != "" {
        os.Setenv("A3K_AUTHOR", *author)
    }

	// Use default kubeconfig if not specified
	if *kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		*kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("Error building kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Execute the command
	err = runCommand(clientset, flag.Args())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(clientset *kubernetes.Clientset, args []string) error {
    if len(args) == 0 {
        showHelp()
        return nil
    }

    switch args[0] {
    case "workloads":
        return getWorkloads(clientset)
    case "nodes":
        return getNodes(clientset)
    case "events":
        return getEvents(clientset)
    case "health":
        return getHealth(clientset)
    case "report":
        return generateReport(clientset)
    case "images":
        return auditImages(clientset)
    case "help", "--help", "-h":
        showHelp()
        return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func showHelp() {
    helpText := `A3K - Assessment · Audit · Analyzer for Kubernetes

Usage:
  a3k [command] [flags]

Available Commands:
  workloads   Show information about workloads (Deployments, StatefulSets, DaemonSets, CronJobs, Pods)
  nodes       Show information about nodes (count, CPU, Memory, machine types)
  events      Show cluster events summary (warnings, reasons, objects)
  health      Show consolidated cluster and workloads health overview
  report      Generate a comprehensive markdown report of the cluster
  images      Audit images (Bitnami vs BitnamiLegacy)
  help        Show this help message

Flags:
  --kubeconfig string   Path to kubeconfig file (default is $HOME/.kube/config)
  --help                Show help message
  -o raw                Output mode; 'raw' disables styling
  --output raw          Same as -o
`
    fmt.Print(helpText)
}
