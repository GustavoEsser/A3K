package main

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterInfo contains information about the Kubernetes cluster
type ClusterInfo struct {
	Provider    string
	Region      string
	K8sVersion  string
	Uptime      string
	NodeCount   int
}

// GetClusterInfo gathers information about the Kubernetes cluster
func GetClusterInfo(clientset *kubernetes.Clientset) (*ClusterInfo, error) {
	info := &ClusterInfo{}

	// Get cluster version
	version, err := clientset.ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("error getting cluster version: %v", err)
	}
	info.K8sVersion = version.GitVersion

	// Get nodes to determine provider and region
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting nodes: %v", err)
	}

	info.NodeCount = len(nodes.Items)

	// If we have nodes, try to determine provider and region
	if len(nodes.Items) > 0 {
		node := nodes.Items[0]
		info.Provider = detectProvider(node)
		info.Region = detectRegion(node)
	}

	// Calculate cluster uptime (simplified - uses the oldest node's creation time)
	if len(nodes.Items) > 0 {
		var oldestNode *corev1.Node
		for i := range nodes.Items {
			if oldestNode == nil || nodes.Items[i].CreationTimestamp.Time.Before(oldestNode.CreationTimestamp.Time) {
				oldestNode = &nodes.Items[i]
			}
		}
		uptime := time.Since(oldestNode.CreationTimestamp.Time)
		days := int(uptime.Hours() / 24)
		hours := int(uptime.Hours()) % 24
		info.Uptime = fmt.Sprintf("%d days, %d hours", days, hours)
	}

	return info, nil
}

// detectProvider detects the cloud provider from node spec
func detectProvider(node corev1.Node) string {
	// Check providerID first
	providerID := node.Spec.ProviderID
	switch {
	case providerID == "":
		// Try to detect from node spec
		if _, ok := node.Labels["node.kubernetes.io/instance-type"]; ok {
			return "AWS (EKS)"
		} else if _, ok := node.Labels["cloud.google.com/gke-nodepool"]; ok {
			return "GCP (GKE)"
		} else if _, ok := node.Labels["kubernetes.azure.com/agentpool"]; ok {
			return "Azure (AKS)"
		}
		return "Unknown (On-premises?)"
	case providerID[:5] == "aws:":
		return "AWS (EKS)"
	case providerID[:4] == "gce:":
		return "GCP (GKE)"
	case providerID[:8] == "azure://":
		return "Azure (AKS)"
	default:
		return "Unknown"
	}
}

// detectRegion detects the region from node spec
func detectRegion(node corev1.Node) string {
	// Check common cloud provider labels
	if region, ok := node.Labels["topology.kubernetes.io/region"]; ok {
		return region
	}
	if region, ok := node.Labels["failure-domain.beta.kubernetes.io/region"]; ok {
		return region
	}
	if zone, ok := node.Labels["topology.kubernetes.io/zone"]; ok {
		// Extract region from zone (e.g., us-central1-a -> us-central1)
		return zone[:len(zone)-2]
	}
	if zone, ok := node.Labels["failure-domain.beta.kubernetes.io/zone"]; ok {
		return zone[:len(zone)-2]
	}

	// Check provider-specific labels
	if region, ok := node.Labels["topology.ebs.csi.aws.com/zone"]; ok {
		return region[:len(region)-1] // Remove the last character (zone letter)
	}

	return "Unknown"
}

// FormatClusterInfo formats cluster information as markdown
func FormatClusterInfo(info *ClusterInfo) string {
	return fmt.Sprintf(`## Cluster Information

| Property        | Value          |
|----------------|---------------|
| Provider       | %-13s |
| Region         | %-13s |
| Kubernetes Ver | %-13s |
| Uptime         | %-13s |
| Node Count     | %-13d |
`,
		info.Provider,
		info.Region,
		info.K8sVersion,
		info.Uptime,
		info.NodeCount,
	)
}
