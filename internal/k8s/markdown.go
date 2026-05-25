package k8s

import "strings"

// EscapeCell sanitizes a value for safe embedding in a Markdown table cell.
func EscapeCell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// BuildMarkdownTable renders a GitHub-flavored markdown table, escaping cell values.
func BuildMarkdownTable(headers []string, rows [][]string) string {
	var sb strings.Builder
	// Header
	sb.WriteString("| ")
	sb.WriteString(strings.Join(headers, " | "))
	sb.WriteString(" |\n")
	// Separator
	sb.WriteString("| ")
	for i := range headers {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString("---")
	}
	sb.WriteString(" |\n")
	// Rows — escape each cell to prevent broken table structure
	for _, r := range rows {
		escaped := make([]string, len(r))
		for i, cell := range r {
			escaped[i] = EscapeCell(cell)
		}
		sb.WriteString("| ")
		sb.WriteString(strings.Join(escaped, " | "))
		sb.WriteString(" |\n")
	}
	return sb.String()
}
