package main

import (
    "fmt"
    "bytes"
    "strings"
    "os"
    "os/exec"
)

// hasGum checks if the gum binary is available in PATH
func hasGum() bool {
    // Allow forcing plain output via environment variable
    if os.Getenv("A3K_NO_GUM") == "1" || os.Getenv("GUM_DISABLE") == "1" {
        return false
    }
    _, err := exec.LookPath("gum")
    return err == nil
}

// styleHeader returns a styled header using gum when available
func styleHeader(text string) string {
    if !hasGum() {
        return fmt.Sprintf("=== %s ===\n", text)
    }
    args := []string{"style", "--border", "double", "--margin", "1 0", "--padding", "0 1", "--align", "center", "--foreground", "#00D7FF", text}
    out, err := exec.Command("gum", args...).Output()
    if err != nil {
        return fmt.Sprintf("=== %s ===\n", text)
    }
    return string(out)
}

// styleSubheader returns a styled subheader using gum when available
func styleSubheader(text string) string {
    if !hasGum() {
        return fmt.Sprintf("%s\n", text)
    }
    args := []string{"style", "--border", "rounded", "--margin", "1 0 0 0", "--padding", "0 1", "--foreground", "#FFD700", text}
    out, err := exec.Command("gum", args...).Output()
    if err != nil {
        return fmt.Sprintf("%s\n", text)
    }
    return string(out)
}

// styleLine returns a styled line using gum when available
func styleLine(text string) string {
    if !hasGum() {
        return text + "\n"
    }
    args := []string{"style", "--foreground", "#A0A0A0", text}
    out, err := exec.Command("gum", args...).Output()
    if err != nil {
        return text + "\n"
    }
    return string(out)
}

// Convenience printers
func printHeader(title string) { fmt.Print(styleHeader(title)) }
func printSubheader(title string) { fmt.Print(styleSubheader(title)) }
func printLine(text string) { fmt.Print(styleLine(text)) }

// printTable renders a table using gum if available, otherwise a plain text table
func printTable(headers []string, rows [][]string) {
    // Build plain text table first for consistent sizing
    plain := buildFallbackTable(headers, rows)
    if hasGum() {
        // Apply a styled wrapper without using gum table to avoid any interactive behavior
        style := exec.Command("gum", "style", "--border", "rounded", "--padding", "0 1", "--margin", "0 0", "--foreground", "#90EE90")
        style.Stdin = bytes.NewBufferString(plain)
        styledOut, err := style.Output()
        if err == nil {
            fmt.Print(string(styledOut))
            return
        }
        // If styling fails, fall back to plain
    }
    fmt.Print(plain)
}

// buildFallbackTable creates a simple aligned table when gum is not available
func buildFallbackTable(headers []string, rows [][]string) string {
    // Compute column widths
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

    // Helper to pad cells
    pad := func(s string, w int) string {
        if len(s) >= w {
            return s
        }
        return s + strings.Repeat(" ", w-len(s))
    }

    var b strings.Builder
    // Header
    for i := 0; i < cols; i++ {
        if i > 0 { b.WriteString("  ") }
        b.WriteString(pad(headers[i], widths[i]))
    }
    b.WriteString("\n")
    // Separator
    for i := 0; i < cols; i++ {
        if i > 0 { b.WriteString("  ") }
        b.WriteString(strings.Repeat("-", widths[i]))
    }
    b.WriteString("\n")
    // Rows
    for _, r := range rows {
        for i := 0; i < cols; i++ {
            if i > 0 { b.WriteString("  ") }
            cell := ""
            if i < len(r) { cell = r[i] }
            b.WriteString(pad(cell, widths[i]))
        }
        b.WriteString("\n")
    }
    return b.String()
}