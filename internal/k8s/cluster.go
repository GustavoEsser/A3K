package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterInfo contains information about the Kubernetes cluster.
type ClusterInfo struct {
	Provider   string
	Region     string
	K8sVersion string
	Uptime     string
	NodeCount  int
}

// GetClusterInfo gathers information about the Kubernetes cluster.
func GetClusterInfo(clientset *kubernetes.Clientset) (*ClusterInfo, error) {
	info := &ClusterInfo{}

	// Get cluster version
	version, err := clientset.ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("error getting cluster version: %w", err)
	}
	info.K8sVersion = version.GitVersion

	// Get nodes to determine provider and region
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting nodes: %w", err)
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

// detectProvider detects the cloud provider from node spec.
func detectProvider(node corev1.Node) string {
	providerID := node.Spec.ProviderID
	switch {
	case providerID == "":
		if _, ok := node.Labels["node.kubernetes.io/instance-type"]; ok {
			return "AWS (EKS)"
		} else if _, ok := node.Labels["cloud.google.com/gke-nodepool"]; ok {
			return "GCP (GKE)"
		} else if _, ok := node.Labels["kubernetes.azure.com/agentpool"]; ok {
			return "Azure (AKS)"
		}
		return "Unknown (On-premises?)"
	case len(providerID) >= 5 && providerID[:5] == "aws:/":
		return "AWS (EKS)"
	case len(providerID) >= 4 && providerID[:4] == "gce:":
		return "GCP (GKE)"
	case len(providerID) >= 8 && providerID[:8] == "azure://":
		return "Azure (AKS)"
	default:
		return "Unknown"
	}
}

// detectRegion detects the region from node spec.
func detectRegion(node corev1.Node) string {
	if region, ok := node.Labels["topology.kubernetes.io/region"]; ok {
		return region
	}
	if region, ok := node.Labels["failure-domain.beta.kubernetes.io/region"]; ok {
		return region
	}
	if zone, ok := node.Labels["topology.kubernetes.io/zone"]; ok {
		if len(zone) > 2 {
			return zone[:len(zone)-2]
		}
		return zone
	}
	if zone, ok := node.Labels["failure-domain.beta.kubernetes.io/zone"]; ok {
		if len(zone) > 2 {
			return zone[:len(zone)-2]
		}
		return zone
	}
	if region, ok := node.Labels["topology.ebs.csi.aws.com/zone"]; ok {
		if len(region) > 1 {
			return region[:len(region)-1]
		}
		return region
	}
	return "Unknown"
}

// FormatClusterInfo formats cluster information as a markdown table (no heading — caller provides it).
func FormatClusterInfo(info *ClusterInfo) string {
	return fmt.Sprintf(`| Property        | Value          |
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
