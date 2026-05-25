package report

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Writer handles secure report file creation.
type Writer struct {
	ReportDir string
}

func NewWriter(reportDir string) *Writer {
	if reportDir == "" {
		home, _ := os.UserHomeDir()
		reportDir = filepath.Join(home, "a3k-reports")
	}
	return &Writer{ReportDir: reportDir}
}

// Save writes content to a timestamped file with secure permissions.
func (w *Writer) Save(content string) (string, error) {
	// 0700: only owner can enter the directory
	if err := os.MkdirAll(w.ReportDir, 0700); err != nil {
		return "", fmt.Errorf("creating reports directory: %w", err)
	}
	ts := time.Now().Format("2006-01-02_15-04-05")
	path := filepath.Join(w.ReportDir, fmt.Sprintf("a3k-report-%s.md", ts))
	// O_EXCL: fail if file already exists; 0600: owner read/write only
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("creating report file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return "", fmt.Errorf("writing report: %w", err)
	}
	return path, nil
}
