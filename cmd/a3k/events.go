package main

import (
    "context"
    "fmt"
    "sort"
    "strings"
    "time"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

// KeyCount holds an aggregated count for a key (reason/object)
type KeyCount struct {
    Key   string
    Count int
}

// EventBrief contains compact info about a warning event
type EventBrief struct {
    Namespace string
    Kind      string
    Name      string
    Reason    string
    Message   string
    Time      time.Time
}

// EventSummary represents a cluster events overview
type EventSummary struct {
    Total             int
    Warnings          int
    Normals           int
    TopWarningReasons []KeyCount
    TopWarningObjects []KeyCount
    RecentWarnings    []EventBrief
}

// AnalyzeClusterEvents collects and summarizes cluster events across namespaces
func AnalyzeClusterEvents(clientset *kubernetes.Clientset) (*EventSummary, error) {
    // List namespaces
    namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return nil, fmt.Errorf("error getting namespaces: %v", err)
    }

    summary := &EventSummary{}

    reasonCounts := make(map[string]int)
    objectCounts := make(map[string]int)
    var warningEvents []EventBrief

    for _, ns := range namespaces.Items {
        events, err := clientset.CoreV1().Events(ns.Name).List(context.TODO(), metav1.ListOptions{})
        if err != nil {
            return nil, fmt.Errorf("error getting events in namespace %s: %v", ns.Name, err)
        }

        for _, e := range events.Items {
            summary.Total++
            if strings.EqualFold(e.Type, "Warning") {
                summary.Warnings++
                reason := e.Reason
                if reason == "" { // fallback
                    reason = "Unknown"
                }
                reasonCounts[reason]++

                // Build object key (namespace may be empty for cluster-scoped objects)
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

    // Top N aggregations
    summary.TopWarningReasons = topN(reasonCounts, 5)
    summary.TopWarningObjects = topN(objectCounts, 5)

    // Recent warnings (last 10 by creation time)
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

// FormatEventSummaryMarkdown formats the event summary as markdown for reports
func FormatEventSummaryMarkdown(s *EventSummary) string {
    if s == nil {
        return ""
    }
    out := "## Events Summary\n\n"
    out += fmt.Sprintf("- Total Events: %d\n", s.Total)
    out += fmt.Sprintf("- Warning Events: %d\n", s.Warnings)
    out += fmt.Sprintf("- Normal Events: %d\n\n", s.Normals)

    out += "### Top Warning Reasons\n"
    if len(s.TopWarningReasons) == 0 {
        out += "(none)\n\n"
    } else {
        for _, r := range s.TopWarningReasons {
            out += fmt.Sprintf("- %s: %d\n", r.Key, r.Count)
        }
        out += "\n"
    }

    out += "### Top Affected Objects (warnings)\n"
    if len(s.TopWarningObjects) == 0 {
        out += "(none)\n\n"
    } else {
        for _, o := range s.TopWarningObjects {
            out += fmt.Sprintf("- %s: %d\n", o.Key, o.Count)
        }
        out += "\n"
    }

    out += "### Recent Warning Events\n"
    if len(s.RecentWarnings) == 0 {
        out += "(none)\n"
    } else {
        for _, e := range s.RecentWarnings {
            out += fmt.Sprintf("- %s | %s/%s %s: %s\n", e.Time.Format(time.RFC3339), e.Namespace, e.Name, e.Kind, e.Reason)
        }
    }
    out += "\n"
    return out
}

// getEvents prints a CLI-friendly overview of cluster events
func getEvents(clientset *kubernetes.Clientset) error {
    summary, err := AnalyzeClusterEvents(clientset)
    if err != nil {
        return err
    }

    printHeader("Cluster Events Overview")
    printTable([]string{"Type", "Count"}, [][]string{
        {"Total Events", fmt.Sprintf("%d", summary.Total)},
        {"Warnings", fmt.Sprintf("%d", summary.Warnings)},
        {"Normals", fmt.Sprintf("%d", summary.Normals)},
    })

    printSubheader("Top Warning Reasons")
    if len(summary.TopWarningReasons) == 0 {
        printLine("(none)")
    } else {
        rows := make([][]string, 0, len(summary.TopWarningReasons))
        for _, r := range summary.TopWarningReasons {
            rows = append(rows, []string{r.Key, fmt.Sprintf("%d", r.Count)})
        }
        printTable([]string{"Reason", "Count"}, rows)
    }

    printSubheader("Top Affected Objects (warnings)")
    if len(summary.TopWarningObjects) == 0 {
        printLine("(none)")
    } else {
        rows := make([][]string, 0, len(summary.TopWarningObjects))
        for _, o := range summary.TopWarningObjects {
            rows = append(rows, []string{o.Key, fmt.Sprintf("%d", o.Count)})
        }
        printTable([]string{"Object", "Count"}, rows)
    }

    printSubheader("Recent Warning Events")
    if len(summary.RecentWarnings) == 0 {
        printLine("(none)")
    } else {
        rows := make([][]string, 0, len(summary.RecentWarnings))
        for _, e := range summary.RecentWarnings {
            msg := e.Message
            if len(msg) > 80 {
                msg = msg[:77] + "..."
            }
            rows = append(rows, []string{e.Time.Format(time.RFC3339), fmt.Sprintf("%s/%s %s", e.Namespace, e.Name, e.Kind), e.Reason, msg})
        }
        printTable([]string{"Time", "Object", "Reason", "Message"}, rows)
    }

    return nil
}