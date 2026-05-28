package k8s

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GenerateIngressesMarkdown lists all ingresses grouped by namespace and returns a Markdown section.
func GenerateIngressesMarkdown(clientset *kubernetes.Clientset) (string, error) {
	ctx := context.Background()
	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting namespaces: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("## Ingresses por Namespace 🌐\n\n")
	sb.WriteString("Esta seção lista todos os Ingresses do cluster, agrupados por namespace, com hosts, TLS e backends.\n\n")

	totalIngresses := 0
	for _, ns := range nsList.Items {
		ings, err := clientset.NetworkingV1().Ingresses(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil || len(ings.Items) == 0 {
			continue
		}

		var rows [][]string
		for _, ing := range ings.Items {
			ingClass := ""
			if ing.Spec.IngressClassName != nil {
				ingClass = *ing.Spec.IngressClassName
			}
			var hosts []string
			for _, r := range ing.Spec.Rules {
				if r.Host != "" {
					hosts = append(hosts, r.Host)
				}
			}
			hostsStr := "-"
			if len(hosts) > 0 {
				hostsStr = strings.Join(hosts, ", ")
			}
			var tlsEntries []string
			for _, t := range ing.Spec.TLS {
				if t.SecretName != "" {
					tlsEntries = append(tlsEntries, t.SecretName)
				}
			}
			tlsStr := "Não"
			if len(tlsEntries) > 0 {
				tlsStr = fmt.Sprintf("Sim (%s)", strings.Join(tlsEntries, ", "))
			}
			var backends []string
			if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
				svc := ing.Spec.DefaultBackend.Service
				port := svc.Port.Number
				if svc.Port.Name != "" {
					backends = append(backends, fmt.Sprintf("%s:%s", svc.Name, svc.Port.Name))
				} else {
					backends = append(backends, fmt.Sprintf("%s:%d", svc.Name, port))
				}
			}
			for _, r := range ing.Spec.Rules {
				if r.HTTP == nil {
					continue
				}
				for _, p := range r.HTTP.Paths {
					if p.Backend.Service != nil {
						svc := p.Backend.Service
						port := svc.Port.Number
						if svc.Port.Name != "" {
							backends = append(backends, fmt.Sprintf("%s:%s (%s)", svc.Name, svc.Port.Name, p.Path))
						} else {
							backends = append(backends, fmt.Sprintf("%s:%d (%s)", svc.Name, port, p.Path))
						}
					}
				}
			}
			backendStr := "-"
			if len(backends) > 0 {
				backendStr = strings.Join(backends, ", ")
			}

			rows = append(rows, []string{ing.Name, ingClass, hostsStr, tlsStr, backendStr})
			totalIngresses++
		}

		if len(rows) > 0 {
			fmt.Fprintf(&sb, "### %s\n\n", ns.Name)
			sb.WriteString(BuildMarkdownTable([]string{"Name", "Class", "Hosts", "TLS", "Backends"}, rows))
			sb.WriteString("\n")
		}
	}

	if totalIngresses == 0 {
		return "", nil
	}
	return sb.String(), nil
}
