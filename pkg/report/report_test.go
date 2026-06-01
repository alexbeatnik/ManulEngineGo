package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/explain"
)

// ---- sanitizeFilename -------------------------------------------------------

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Login Flow", "login_flow"},
		{"UPPER CASE", "upper_case"},
		{"hello-world", "hello-world"},
		{"hello_world", "hello_world"},
		{"special!@#chars", "specialchars"},
		{"mixed 123 !test", "mixed_123_test"},
		{"", "untitled"},
		{"   ", "untitled"},
		{"!!!###", "untitled"},
		{"abc", "abc"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := sanitizeFilename(tc.in)
			if got != tc.want {
				t.Errorf("sanitizeFilename(%q) = %q want %q", tc.in, got, tc.want)
			}
		})
	}
}

// ---- GenerateHTML -----------------------------------------------------------

func makeResult(title, file string, success bool) *explain.HuntResult {
	return &explain.HuntResult{
		Title:           title,
		HuntFile:        file,
		TotalSteps:      2,
		Passed:          1,
		Failed:          1,
		TotalDurationMS: 500,
		Success:         success,
		Results: []explain.ExecutionResult{
			{Step: "Click Login", StepIndex: 0, Success: true, DurationMS: 200},
			{Step: "Fill password", StepIndex: 1, Success: false, DurationMS: 300, Error: "element not found"},
		},
	}
}

func TestGenerateHTML_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	result := makeResult("Login Flow", "login.hunt", true)

	path, err := GenerateHTML(result, dir)
	if err != nil {
		t.Fatalf("GenerateHTML: %v", err)
	}
	if !strings.HasSuffix(path, ".html") {
		t.Errorf("path %q does not end with .html", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestGenerateHTML_ContainsStepText(t *testing.T) {
	dir := t.TempDir()
	result := makeResult("Smoke Test", "smoke.hunt", false)

	path, err := GenerateHTML(result, dir)
	if err != nil {
		t.Fatalf("GenerateHTML: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)

	for _, needle := range []string{
		"ManulHeart Report",
		"Smoke Test",
		"smoke.hunt",
		"Click Login",
		"Fill password",
		"element not found",
	} {
		if !strings.Contains(content, needle) {
			t.Errorf("report missing %q", needle)
		}
	}
}

func TestGenerateHTML_TitleEmpty_UsesUntitled(t *testing.T) {
	dir := t.TempDir()
	result := makeResult("", "smoke.hunt", true)

	path, err := GenerateHTML(result, dir)
	if err != nil {
		t.Fatalf("GenerateHTML: %v", err)
	}
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "report_untitled_") {
		t.Errorf("filename %q should start with report_untitled_", base)
	}
}

func TestGenerateHTML_SoftErrors(t *testing.T) {
	dir := t.TempDir()
	result := makeResult("Soft", "soft.hunt", true)
	result.SoftErrors = []string{"warn1", "warn2"}

	path, err := GenerateHTML(result, dir)
	if err != nil {
		t.Fatalf("GenerateHTML: %v", err)
	}
	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "warn1") || !strings.Contains(content, "warn2") {
		t.Errorf("soft errors missing from report")
	}
}

func TestGenerateHTML_CreatesOutDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	result := makeResult("T", "t.hunt", true)

	_, err := GenerateHTML(result, nested)
	if err != nil {
		t.Fatalf("GenerateHTML with nested outDir: %v", err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Errorf("nested dir not created: %v", err)
	}
}

func TestGenerateHTML_UniqueFilenames(t *testing.T) {
	dir := t.TempDir()
	result := makeResult("Same Title", "x.hunt", true)

	p1, err1 := GenerateHTML(result, dir)
	p2, err2 := GenerateHTML(result, dir)
	if err1 != nil || err2 != nil {
		t.Fatalf("GenerateHTML errors: %v, %v", err1, err2)
	}
	if p1 == p2 {
		t.Errorf("expected unique filenames for parallel calls, got %q twice", p1)
	}
}

// ---- GenerateIndex ----------------------------------------------------------

func TestGenerateIndex_CreatesIndexHTML(t *testing.T) {
	dir := t.TempDir()
	summaries := []RunSummary{
		{Result: makeResult("Hunt A", "a.hunt", true), WorkerID: 1},
		{Result: makeResult("Hunt B", "b.hunt", false), WorkerID: 2},
	}

	path, err := GenerateIndex(summaries, dir)
	if err != nil {
		t.Fatalf("GenerateIndex: %v", err)
	}
	if filepath.Base(path) != "index.html" {
		t.Errorf("path %q should end in index.html", path)
	}
}

func TestGenerateIndex_NilResult(t *testing.T) {
	dir := t.TempDir()
	summaries := []RunSummary{
		{Result: nil, WorkerID: 1},
		{Result: makeResult("Good", "good.hunt", true), WorkerID: 2},
	}

	path, err := GenerateIndex(summaries, dir)
	if err != nil {
		t.Fatalf("GenerateIndex: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "missing result") {
		t.Errorf("nil result row should contain 'missing result'")
	}
}

func TestGenerateIndex_Aggregates(t *testing.T) {
	dir := t.TempDir()

	r1 := makeResult("Hunt A", "a.hunt", true)
	r1.TotalSteps = 3
	r1.Passed = 3
	r1.Failed = 0
	r1.TotalDurationMS = 100

	r2 := makeResult("Hunt B", "b.hunt", false)
	r2.TotalSteps = 2
	r2.Passed = 1
	r2.Failed = 1
	r2.TotalDurationMS = 200

	summaries := []RunSummary{
		{Result: r1, WorkerID: 1},
		{Result: r2, WorkerID: 2},
	}

	path, err := GenerateIndex(summaries, dir)
	if err != nil {
		t.Fatalf("GenerateIndex: %v", err)
	}
	data, _ := os.ReadFile(path)
	content := string(data)

	// overall badge: fail because r2 failed
	if !strings.Contains(content, "FAIL") {
		t.Error("expected FAIL badge when any hunt failed")
	}
	// both hunt titles present
	if !strings.Contains(content, "Hunt A") {
		t.Error("Hunt A missing from index")
	}
	if !strings.Contains(content, "Hunt B") {
		t.Error("Hunt B missing from index")
	}
}

func TestGenerateIndex_AllPass(t *testing.T) {
	dir := t.TempDir()
	summaries := []RunSummary{
		{Result: makeResult("A", "a.hunt", true), WorkerID: 1},
		{Result: makeResult("B", "b.hunt", true), WorkerID: 2},
	}

	path, err := GenerateIndex(summaries, dir)
	if err != nil {
		t.Fatalf("GenerateIndex: %v", err)
	}
	data, _ := os.ReadFile(path)
	content := string(data)
	// Overall badge should be PASS when all succeed.
	if !strings.Contains(content, `badge pass`) {
		t.Error("expected pass badge when all hunts passed")
	}
}

func TestGenerateIndex_ReportLink(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "report_a_20260101_000001_000001.html")
	// create the file so it exists
	_ = os.WriteFile(reportPath, []byte(""), 0o644)

	summaries := []RunSummary{
		{Result: makeResult("A", "a.hunt", true), ReportPath: reportPath, WorkerID: 1},
	}

	path, err := GenerateIndex(summaries, dir)
	if err != nil {
		t.Fatalf("GenerateIndex: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "view") {
		t.Error("expected report link 'view' in index for non-empty ReportPath")
	}
}

func TestGenerateIndex_CreatesOutDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested", "reports")

	_, err := GenerateIndex(nil, nested)
	if err != nil {
		t.Fatalf("GenerateIndex with nested dir: %v", err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Errorf("nested dir not created: %v", err)
	}
}
