package output

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// TableRenderer renders output using gum when available, falling back to plain text.
type TableRenderer struct {
	noColor bool
	w       io.Writer
}

func NewTableRenderer(noColor bool) *TableRenderer {
	return &TableRenderer{noColor: noColor, w: os.Stdout}
}

func (r *TableRenderer) Writer() io.Writer { return r.w }

func (r *TableRenderer) hasGum() bool {
	if r.noColor || os.Getenv("A3K_NO_GUM") == "1" || os.Getenv("GUM_DISABLE") == "1" {
		return false
	}
	_, err := exec.LookPath("gum")
	return err == nil
}

func (r *TableRenderer) Header(title string) {
	if !r.hasGum() {
		fmt.Fprintf(r.w, "\n=== %s ===\n\n", title)
		return
	}
	args := []string{"style", "--border", "double", "--margin", "1 0", "--padding", "0 1", "--align", "center", "--foreground", "#00D7FF", title}
	out, err := exec.Command("gum", args...).Output()
	if err != nil {
		fmt.Fprintf(r.w, "\n=== %s ===\n\n", title)
		return
	}
	fmt.Fprint(r.w, string(out))
}

func (r *TableRenderer) Subheader(title string) {
	if !r.hasGum() {
		fmt.Fprintf(r.w, "\n%s\n", title)
		return
	}
	args := []string{"style", "--border", "rounded", "--margin", "1 0 0 0", "--padding", "0 1", "--foreground", "#FFD700", title}
	out, err := exec.Command("gum", args...).Output()
	if err != nil {
		fmt.Fprintf(r.w, "\n%s\n", title)
		return
	}
	fmt.Fprint(r.w, string(out))
}

func (r *TableRenderer) Line(text string) {
	if !r.hasGum() {
		fmt.Fprintln(r.w, text)
		return
	}
	args := []string{"style", "--foreground", "#A0A0A0", text}
	out, err := exec.Command("gum", args...).Output()
	if err != nil {
		fmt.Fprintln(r.w, text)
		return
	}
	fmt.Fprint(r.w, string(out))
}

func (r *TableRenderer) Print(text string) {
	fmt.Fprint(r.w, text)
}

func (r *TableRenderer) Table(headers []string, rows [][]string) {
	plain := buildPlainTable(headers, rows)
	if r.hasGum() {
		style := exec.Command("gum", "style", "--border", "rounded", "--padding", "0 1", "--margin", "0 0", "--foreground", "#90EE90")
		style.Stdin = bytes.NewBufferString(plain)
		if out, err := style.Output(); err == nil {
			fmt.Fprint(r.w, string(out))
			return
		}
	}
	fmt.Fprint(r.w, plain)
}

func buildPlainTable(headers []string, rows [][]string) string {
	cols := len(headers)
	widths := make([]int, cols)
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range rows {
		for i := 0; i < cols && i < len(r); i++ {
			if len(r[i]) > widths[i] {
				widths[i] = len(r[i])
			}
		}
	}
	pad := func(s string, w int) string {
		if len(s) >= w {
			return s
		}
		return s + strings.Repeat(" ", w-len(s))
	}
	var b strings.Builder
	for i, h := range headers {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(pad(h, widths[i]))
	}
	b.WriteString("\n")
	for i, w := range widths {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(strings.Repeat("-", w))
	}
	b.WriteString("\n")
	for _, row := range rows {
		for i := 0; i < cols; i++ {
			if i > 0 {
				b.WriteString("  ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			b.WriteString(pad(cell, widths[i]))
		}
		b.WriteString("\n")
	}
	return b.String()
}
