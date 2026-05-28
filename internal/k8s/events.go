package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// KeyCount holds an aggregated count for a key (reason/object).
type KeyCount struct {
	Key   string
	Count int
}

// EventBrief contains compact info about a warning event.
type EventBrief struct {
	Namespace string
	Kind      string
	Name      string
	Reason    string
	Message   string
	Time      time.Time
}

// EventSummary represents a cluster events overview.
type EventSummary struct {
	Total             int
	Warnings          int
	Normals           int
	TopWarningReasons []KeyCount
	TopWarningObjects []KeyCount
	RecentWarnings    []EventBrief
}

// AnalyzeClusterEvents collects and summarizes cluster events across namespaces.
func AnalyzeClusterEvents(clientset *kubernetes.Clientset) (*EventSummary, error) {
	ctx := context.Background()

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting namespaces: %w", err)
	}

	summary := &EventSummary{}
	reasonCounts := make(map[string]int)
	objectCounts := make(map[string]int)
	var warningEvents []EventBrief

	for _, ns := range namespaces.Items {
		events, err := clientset.CoreV1().Events(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting events in namespace %s: %w", ns.Name, err)
		}

		for _, e := range events.Items {
			summary.Total++
			if strings.EqualFold(e.Type, "Warning") {
				summary.Warnings++
				reason := e.Reason
				if reason == "" {
					reason = "Unknown"
				}
				reasonCounts[reason]++

				objNS := ns.Name
				if e.InvolvedObject.Namespace != "" {
					objNS = e.InvolvedObject.Namespace
				}
				objKey := fmt.Sprintf("%s/%s %s", objNS, e.InvolvedObject.Name, e.InvolvedObject.Kind)
				objectCounts[objKey]++

				warningEvents = append(warningEvents, EventBrief{
					Namespace: objNS,
					Kind:      e.InvolvedObject.Kind,
					Name:      e.InvolvedObject.Name,
					Reason:    reason,
					Message:   e.Message,
					Time:      e.CreationTimestamp.Time,
				})
			} else {
				summary.Normals++
			}
		}
	}

	summary.TopWarningReasons = topN(reasonCounts, 5)
	summary.TopWarningObjects = topN(objectCounts, 5)

	sort.Slice(warningEvents, func(i, j int) bool { return warningEvents[i].Time.After(warningEvents[j].Time) })
	if len(warningEvents) > 10 {
		summary.RecentWarnings = warningEvents[:10]
	} else {
		summary.RecentWarnings = warningEvents
	}

	return summary, nil
}

func topN(m map[string]int, n int) []KeyCount {
	items := make([]KeyCount, 0, len(m))
	for k, v := range m {
		items = append(items, KeyCount{Key: k, Count: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key < items[j].Key
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > n {
		return items[:n]
	}
	return items
}

// FormatEventSummaryMarkdown formats the event summary as markdown for reports.
func FormatEventSummaryMarkdown(s *EventSummary) string {
	if s == nil {
		return ""
	}
	out := "Os eventos no Kubernetes são registros de ocorrências que descrevem mudanças de estado, alertas e informações relevantes sobre objetos do cluster, como pods, nós e deployments. Eles funcionam como um histórico de diagnósticos em tempo real, ajudando a identificar problemas, acompanhar falhas de execução e entender o comportamento dos workloads ao longo do tempo.\n\n" //nolint:misspell // Portuguese
	out += fmt.Sprintf("- Total Events: %d\n", s.Total)
	out += fmt.Sprintf("- Warning Events: %d\n", s.Warnings)
	out += fmt.Sprintf("- Normal Events: %d\n\n", s.Normals)

	out += "### Principais Motivos de Warning\n"
	if len(s.TopWarningReasons) == 0 {
		out += "(nenhum)\n\n"
	} else {
		for _, r := range s.TopWarningReasons {
			out += fmt.Sprintf("- %s: %d\n", r.Key, r.Count)
		}
		out += "\n"
	}

	out += "### Objetos Mais Afetados (warnings)\n"
	if len(s.TopWarningObjects) == 0 {
		out += "(nenhum)\n\n"
	} else {
		for _, o := range s.TopWarningObjects {
			out += fmt.Sprintf("- %s: %d\n", o.Key, o.Count)
		}
		out += "\n"
	}

	out += "### Eventos de Warning Recentes\n"
	if len(s.RecentWarnings) == 0 {
		out += "(nenhum)\n"
	} else {
		for _, e := range s.RecentWarnings {
			out += fmt.Sprintf("- %s | %s/%s %s: %s\n", e.Time.Format(time.RFC3339), e.Namespace, e.Name, e.Kind, e.Reason)
		}
	}
	out += "\n"
	return out
}
