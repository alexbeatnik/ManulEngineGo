package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alexbeatnik/ManulHeart/pkg/explain"
)

// RunHistoryEntry is one line of reports/run_history.json per the extension contract.
type RunHistoryEntry struct {
	File       string  `json:"file"`
	Name       string  `json:"name"`
	Timestamp  string  `json:"timestamp"`
	Status     string  `json:"status"`
	DurationMS float64 `json:"duration_ms"`
}

// AppendRunHistory appends one JSONL record describing result to
// <reportsDir>/run_history.json (created if needed). The file is append-only;
// each record ends with a newline. Errors are wrapped and returned so the
// caller can surface them without failing the run.
func AppendRunHistory(reportsDir string, result *explain.HuntResult) error {
	if result == nil {
		return fmt.Errorf("nil HuntResult")
	}
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		return fmt.Errorf("create reports dir: %w", err)
	}

	absFile, err := filepath.Abs(result.HuntFile)
	if err != nil {
		absFile = result.HuntFile
	}

	status := "fail"
	if result.Success {
		status = "pass"
		if len(result.SoftErrors) > 0 {
			status = "warning"
		}
	}

	entry := RunHistoryEntry{
		File:       absFile,
		Name:       filepath.Base(result.HuntFile),
		Timestamp:  time.Now().UTC().Format("2006-01-02T15:04:05.000000+00:00"),
		Status:     status,
		DurationMS: float64(result.TotalDurationMS),
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal run_history entry: %w", err)
	}

	path := filepath.Join(reportsDir, "run_history.json")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open run_history: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write run_history: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync run_history: %w", err)
	}
	return nil
}
