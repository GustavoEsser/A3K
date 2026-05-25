package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// JSONRenderer renders output as JSON.
type JSONRenderer struct {
	w io.Writer
}

func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{w: os.Stdout}
}

func (r *JSONRenderer) Writer() io.Writer      { return r.w }
func (r *JSONRenderer) Header(title string)    {}
func (r *JSONRenderer) Subheader(title string) {}
func (r *JSONRenderer) Line(text string)       { fmt.Fprintln(r.w, text) }
func (r *JSONRenderer) Print(text string)      { fmt.Fprint(r.w, text) }

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

// RawRenderer is plain text with no decoration.
type RawRenderer struct {
	w io.Writer
}

func NewRawRenderer() *RawRenderer {
	return &RawRenderer{w: os.Stdout}
}

func (r *RawRenderer) Writer() io.Writer      { return r.w }
func (r *RawRenderer) Header(title string)    { fmt.Fprintf(r.w, "\n=== %s ===\n\n", title) }
func (r *RawRenderer) Subheader(title string) { fmt.Fprintf(r.w, "\n%s\n", title) }
func (r *RawRenderer) Line(text string)       { fmt.Fprintln(r.w, text) }
func (r *RawRenderer) Print(text string)      { fmt.Fprint(r.w, text) }
func (r *RawRenderer) Table(headers []string, rows [][]string) {
	fmt.Fprint(r.w, buildPlainTable(headers, rows))
}
