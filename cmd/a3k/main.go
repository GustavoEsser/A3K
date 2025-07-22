package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Initialize command line flags
	kubeconfig := flag.String("kubeconfig", "", "path to the kubeconfig file (default is $HOME/.kube/config)")
	help := flag.Bool("help", false, "show help message")
	flag.Parse()

	if *help {
		showHelp()
		os.Exit(0)
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
	case "report":
		return generateReport(clientset)
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
  report      Generate a comprehensive markdown report of the cluster
  help        Show this help message

Flags:
  --kubeconfig string   Path to kubeconfig file (default is $HOME/.kube/config)
  --help                Show help message
`
	fmt.Print(helpText)
}
