// Package report generates HTML execution reports for ManulEngine (Go) test runs.
package report

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alexbeatnik/ManulEngineGo/pkg/explain"
)

// reportSeq is a monotonic counter appended to every generated filename so
// parallel workers writing reports for hunts with the same title in the
// same second do not silently overwrite each other.
var reportSeq atomic.Uint64

// GenerateHTML writes an HTML report file for the given HuntResult.
// Returns the path to the written report file.
func GenerateHTML(result *explain.HuntResult, outDir string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("create report dir: %w", err)
	}

	filename := fmt.Sprintf("report_%s_%s_%06d.html",
		sanitizeFilename(result.Title),
		time.Now().Format("20060102_150405"),
		reportSeq.Add(1))
	outPath := filepath.Join(outDir, filename)

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>ManulEngine (Go) Report</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
         background: #1a1a2e; color: #e0e0e0; padding: 2rem; }
  h1 { color: #ff9f43; margin-bottom: 0.5rem; }
  .meta { color: #888; margin-bottom: 1.5rem; }
  .summary { display: flex; gap: 2rem; margin-bottom: 2rem; }
  .stat { background: #16213e; border-radius: 8px; padding: 1rem 1.5rem; min-width: 120px; text-align: center; }
  .stat .num { font-size: 2rem; font-weight: bold; }
  .stat.pass .num { color: #2ecc71; }
  .stat.fail .num { color: #e74c3c; }
  .stat.total .num { color: #3498db; }
  .stat.time .num { color: #f1c40f; }
  .result-ok { background: #16213e; border-left: 3px solid #2ecc71; }
  .result-fail { background: #2a1a1a; border-left: 3px solid #e74c3c; }
  .result { margin-bottom: 0.5rem; padding: 0.6rem 1rem; border-radius: 4px; font-family: monospace; font-size: 0.9rem; }
  .result .idx { color: #888; margin-right: 0.5rem; }
  .result .time { color: #888; float: right; }
  .result .err { color: #e74c3c; display: block; margin-top: 0.3rem; padding-left: 2rem; }
  .screenshot { margin-top: 0.3rem; }
  .screenshot img { max-width: 400px; border: 1px solid #333; border-radius: 4px; }
  .badge { display: inline-block; padding: 0.2rem 0.6rem; border-radius: 4px; font-weight: bold; font-size: 1.2rem; }
  .badge.pass { background: #2ecc71; color: #000; }
  .badge.fail { background: #e74c3c; color: #fff; }
  .soft-errors { background: #2a2a1a; border-left: 3px solid #f1c40f; padding: 0.6rem 1rem; margin-top: 1rem; border-radius: 4px; }
  .soft-errors h3 { color: #f1c40f; margin-bottom: 0.3rem; }
</style>
</head>
<body>
`)

	// Header
	badge := "pass"
	if !result.Success {
		badge = "fail"
	}
	b.WriteString(fmt.Sprintf("<h1>ManulEngine (Go) Report <span class=\"badge %s\">%s</span></h1>\n",
		badge, strings.ToUpper(badge)))

	if result.Title != "" {
		b.WriteString(fmt.Sprintf("<div class=\"meta\">%s</div>\n", html.EscapeString(result.Title)))
	}
	if result.HuntFile != "" {
		b.WriteString(fmt.Sprintf("<div class=\"meta\">File: %s</div>\n", html.EscapeString(result.HuntFile)))
	}

	// Summary stats
	b.WriteString("<div class=\"summary\">\n")
	b.WriteString(fmt.Sprintf("<div class=\"stat total\"><div class=\"num\">%d</div>Total</div>\n", result.TotalSteps))
	b.WriteString(fmt.Sprintf("<div class=\"stat pass\"><div class=\"num\">%d</div>Passed</div>\n", result.Passed))
	b.WriteString(fmt.Sprintf("<div class=\"stat fail\"><div class=\"num\">%d</div>Failed</div>\n", result.Failed))
	b.WriteString(fmt.Sprintf("<div class=\"stat time\"><div class=\"num\">%.1fs</div>Duration</div>\n",
		result.TotalDuration.Seconds()))
	b.WriteString("</div>\n")

	// Step results
	for _, r := range result.Results {
		cls := "result-ok"
		if !r.Success {
			cls = "result-fail"
		}
		b.WriteString(fmt.Sprintf("<div class=\"result %s\">\n", cls))
		b.WriteString(fmt.Sprintf("  <span class=\"idx\">[%d]</span> %s", r.StepIndex+1, html.EscapeString(r.Step)))
		b.WriteString(fmt.Sprintf("  <span class=\"time\">%dms</span>\n", r.DurationMS))
		if r.Error != "" {
			b.WriteString(fmt.Sprintf("  <span class=\"err\">%s</span>\n", html.EscapeString(r.Error)))
		}
		if r.ScreenshotPath != "" {
			b.WriteString(fmt.Sprintf("  <div class=\"screenshot\"><img src=\"../%s\" alt=\"screenshot\"></div>\n",
				html.EscapeString(r.ScreenshotPath)))
		}
		b.WriteString("</div>\n")
	}

	// Soft errors
	if len(result.SoftErrors) > 0 {
		b.WriteString("<div class=\"soft-errors\">\n<h3>Soft Verify Warnings</h3>\n<ul>\n")
		for _, se := range result.SoftErrors {
			b.WriteString(fmt.Sprintf("<li>%s</li>\n", html.EscapeString(se)))
		}
		b.WriteString("</ul>\n</div>\n")
	}

	b.WriteString("</body>\n</html>\n")

	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return outPath, nil
}

func sanitizeFilename(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		if r == ' ' {
			return '_'
		}
		return -1
	}, s)
	if s == "" {
		s = "untitled"
	}
	return s
}
