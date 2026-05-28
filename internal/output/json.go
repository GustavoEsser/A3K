package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// JSONRenderer renders output silently — used when --output=json.
type JSONRenderer struct {
	w io.Writer
}

// NewJSONRenderer creates a JSONRenderer writing to stdout.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{w: os.Stdout}
}

// Writer returns the underlying io.Writer.
func (r *JSONRenderer) Writer() io.Writer { return r.w }

// Header is a no-op for JSON output.
func (r *JSONRenderer) Header(_ string) {}

// Subheader is a no-op for JSON output.
func (r *JSONRenderer) Subheader(_ string) {}

// Line writes a plain text line.
func (r *JSONRenderer) Line(text string) { _, _ = fmt.Fprintln(r.w, text) }

// Print writes text without a trailing newline.
func (r *JSONRenderer) Print(text string) { _, _ = fmt.Fprint(r.w, text) }

// Table encodes rows as a JSON array of objects keyed by header names.
func (r *JSONRenderer) Table(headers []string, rows [][]string) {
	records := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		rec := make(map[string]string)
		for i, h := range headers {
			if i < len(row) {
				rec[h] = row[i]
			}
		}
		records = append(records, rec)
	}
	enc := json.NewEncoder(r.w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(records)
}

// RawRenderer renders output as plain text with no decoration.
type RawRenderer struct {
	w io.Writer
}

// NewRawRenderer creates a RawRenderer writing to stdout.
func NewRawRenderer() *RawRenderer {
	return &RawRenderer{w: os.Stdout}
}

// Writer returns the underlying io.Writer.
func (r *RawRenderer) Writer() io.Writer { return r.w }

// Header writes a plain section heading.
func (r *RawRenderer) Header(title string) { _, _ = fmt.Fprintf(r.w, "\n=== %s ===\n\n", title) }

// Subheader writes a plain subsection heading.
func (r *RawRenderer) Subheader(title string) { _, _ = fmt.Fprintf(r.w, "\n%s\n", title) }

// Line writes a plain text line.
func (r *RawRenderer) Line(text string) { _, _ = fmt.Fprintln(r.w, text) }

// Print writes text without a trailing newline.
func (r *RawRenderer) Print(text string) { _, _ = fmt.Fprint(r.w, text) }

// Table renders a plain text table.
func (r *RawRenderer) Table(headers []string, rows [][]string) {
	_, _ = fmt.Fprint(r.w, buildPlainTable(headers, rows))
}
