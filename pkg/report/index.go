package report

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexbeatnik/ManulEngineGo/pkg/explain"
)

// RunSummary is one row in an aggregate report — typically one parallel
// worker's hunt outcome.
type RunSummary struct {
	// Result is the hunt result. Required.
	Result *explain.HuntResult
	// ReportPath is the relative or absolute path to the per-hunt HTML
	// report (the path returned by GenerateHTML). May be empty if no
	// per-hunt report was generated.
	ReportPath string
	// WorkerID identifies the worker that ran this hunt (0 = unspecified).
	WorkerID int
}

// GenerateIndex writes an aggregate index.html summarising every hunt in
// summaries. Intended for parallel runs where one top-level artifact is
// preferable over scanning per-hunt files. Returns the path to index.html.
func GenerateIndex(summaries []RunSummary, outDir string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("create report dir: %w", err)
	}

	totalHunts := len(summaries)
	totalPassed, totalFailed := 0, 0
	totalSteps, totalStepsPassed, totalStepsFailed := 0, 0, 0
	var totalDurationMS int64
	for _, s := range summaries {
		if s.Result == nil {
			continue
		}
		if s.Result.Success {
			totalPassed++
		} else {
			totalFailed++
		}
		totalSteps += s.Result.TotalSteps
		totalStepsPassed += s.Result.Passed
		totalStepsFailed += s.Result.Failed
		totalDurationMS += s.Result.TotalDurationMS
	}

	overallBadge := "pass"
	if totalFailed > 0 {
		overallBadge = "fail"
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>ManulEngine (Go) Run Report</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
         background: #1a1a2e; color: #e0e0e0; padding: 2rem; }
  h1 { color: #ff9f43; margin-bottom: 0.5rem; }
  .meta { color: #888; margin-bottom: 1.5rem; }
  .summary { display: flex; gap: 2rem; margin-bottom: 2rem; flex-wrap: wrap; }
  .stat { background: #16213e; border-radius: 8px; padding: 1rem 1.5rem; min-width: 120px; text-align: center; }
  .stat .num { font-size: 2rem; font-weight: bold; }
  .stat.pass .num { color: #2ecc71; }
  .stat.fail .num { color: #e74c3c; }
  .stat.total .num { color: #3498db; }
  .stat.time .num { color: #f1c40f; }
  table { width: 100%; border-collapse: collapse; background: #16213e; border-radius: 8px; overflow: hidden; }
  th, td { padding: 0.6rem 1rem; text-align: left; border-bottom: 1px solid #1f2a44; }
  th { background: #0f1a30; color: #ff9f43; font-weight: 600; }
  tr.fail td { background: #2a1a1a; }
  tr.pass td { background: #16213e; }
  td.outcome { font-weight: bold; }
  td.outcome.pass { color: #2ecc71; }
  td.outcome.fail { color: #e74c3c; }
  a { color: #3498db; text-decoration: none; }
  a:hover { text-decoration: underline; }
  .badge { display: inline-block; padding: 0.2rem 0.6rem; border-radius: 4px; font-weight: bold; font-size: 1.2rem; }
  .badge.pass { background: #2ecc71; color: #000; }
  .badge.fail { background: #e74c3c; color: #fff; }
</style>
</head>
<body>
`)

	b.WriteString(fmt.Sprintf("<h1>ManulEngine (Go) Run Report <span class=\"badge %s\">%s</span></h1>\n",
		overallBadge, strings.ToUpper(overallBadge)))
	b.WriteString(fmt.Sprintf("<div class=\"meta\">Generated %s</div>\n",
		html.EscapeString(time.Now().Format(time.RFC3339))))

	b.WriteString("<div class=\"summary\">\n")
	b.WriteString(fmt.Sprintf("<div class=\"stat total\"><div class=\"num\">%d</div>Hunts</div>\n", totalHunts))
	b.WriteString(fmt.Sprintf("<div class=\"stat pass\"><div class=\"num\">%d</div>Passed</div>\n", totalPassed))
	b.WriteString(fmt.Sprintf("<div class=\"stat fail\"><div class=\"num\">%d</div>Failed</div>\n", totalFailed))
	b.WriteString(fmt.Sprintf("<div class=\"stat total\"><div class=\"num\">%d</div>Steps</div>\n", totalSteps))
	b.WriteString(fmt.Sprintf("<div class=\"stat pass\"><div class=\"num\">%d</div>Steps Passed</div>\n", totalStepsPassed))
	b.WriteString(fmt.Sprintf("<div class=\"stat fail\"><div class=\"num\">%d</div>Steps Failed</div>\n", totalStepsFailed))
	b.WriteString(fmt.Sprintf("<div class=\"stat time\"><div class=\"num\">%.1fs</div>Total</div>\n",
		float64(totalDurationMS)/1000.0))
	b.WriteString("</div>\n")

	b.WriteString("<table>\n<thead><tr>")
	b.WriteString("<th>#</th><th>Title</th><th>File</th><th>Worker</th>")
	b.WriteString("<th>Steps</th><th>Passed</th><th>Failed</th><th>Duration</th><th>Outcome</th><th>Report</th>")
	b.WriteString("</tr></thead>\n<tbody>\n")

	for i, s := range summaries {
		if s.Result == nil {
			b.WriteString(fmt.Sprintf("<tr class=\"fail\"><td>%d</td><td colspan=\"9\">missing result</td></tr>\n", i+1))
			continue
		}
		cls := "pass"
		outcome := "PASS"
		if !s.Result.Success {
			cls = "fail"
			outcome = "FAIL"
		}
		title := s.Result.Title
		if title == "" {
			title = "(untitled)"
		}
		reportLink := ""
		if s.ReportPath != "" {
			rel, err := filepath.Rel(outDir, s.ReportPath)
			if err != nil {
				rel = s.ReportPath
			}
			rel = filepath.ToSlash(rel)
			reportLink = fmt.Sprintf("<a href=\"%s\">view</a>", html.EscapeString(rel))
		}
		b.WriteString(fmt.Sprintf("<tr class=\"%s\">", cls))
		b.WriteString(fmt.Sprintf("<td>%d</td><td>%s</td><td>%s</td><td>w%d</td>",
			i+1,
			html.EscapeString(title),
			html.EscapeString(s.Result.HuntFile),
			s.WorkerID))
		b.WriteString(fmt.Sprintf("<td>%d</td><td>%d</td><td>%d</td><td>%dms</td>",
			s.Result.TotalSteps, s.Result.Passed, s.Result.Failed, s.Result.TotalDurationMS))
		b.WriteString(fmt.Sprintf("<td class=\"outcome %s\">%s</td><td>%s</td>", cls, outcome, reportLink))
		b.WriteString("</tr>\n")
	}

	b.WriteString("</tbody>\n</table>\n</body>\n</html>\n")

	indexPath := filepath.Join(outDir, "index.html")
	if err := os.WriteFile(indexPath, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write index: %w", err)
	}
	return indexPath, nil
}
