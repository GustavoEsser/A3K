package output

import "io"

// Renderer is the interface all output formats must implement.
type Renderer interface {
	Header(title string)
	Subheader(title string)
	Table(headers []string, rows [][]string)
	Line(text string)
	Print(text string)
	Writer() io.Writer
}

// New returns a Renderer for the given format ("table", "raw", "json").
// Falls back to TableRenderer for unknown formats.
func New(format string, noColor bool) Renderer {
	switch format {
	case "json":
		return NewJSONRenderer()
	case "raw":
		return NewRawRenderer()
	default:
		return NewTableRenderer(noColor)
	}
}
