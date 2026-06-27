package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---- splitCSV ---------------------------------------------------------------

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"  a , b , c ", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"a,,b", []string{"a", "b"}},
		{"", []string{}},
		{"  ,  ,  ", []string{}},
	}
	for _, tc := range cases {
		got := splitCSV(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("splitCSV(%q): len=%d want %d", tc.in, len(got), len(tc.want))
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitCSV(%q)[%d]=%q want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

// ---- Default() --------------------------------------------------------------

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Browser != "chromium" {
		t.Errorf("Browser=%q want %q", cfg.Browser, "chromium")
	}
	if cfg.Headless != false {
		t.Error("Headless should default to false")
	}
	if cfg.DefaultTimeout != 5000*time.Millisecond {
		t.Errorf("DefaultTimeout=%v want 5000ms", cfg.DefaultTimeout)
	}
	if cfg.NavTimeout != 30000*time.Millisecond {
		t.Errorf("NavTimeout=%v want 30000ms", cfg.NavTimeout)
	}
	if cfg.Screenshot != "on-fail" {
		t.Errorf("Screenshot=%q want %q", cfg.Screenshot, "on-fail")
	}
	if cfg.HTMLReport != false {
		t.Error("HTMLReport should default to false")
	}
	if cfg.Retries != 0 {
		t.Errorf("Retries=%d want 0", cfg.Retries)
	}
	if cfg.VerifyMaxRetries != 15 {
		t.Errorf("VerifyMaxRetries=%d want 15", cfg.VerifyMaxRetries)
	}
	if cfg.Workers != 1 {
		t.Errorf("Workers=%d want 1", cfg.Workers)
	}
	if cfg.TestsHome != "tests" {
		t.Errorf("TestsHome=%q want %q", cfg.TestsHome, "tests")
	}
}

// ---- applyJSONFile ----------------------------------------------------------

func writeJSON(t *testing.T, dir string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manul_engine_configuration.json"), data, 0o644); err != nil {
		t.Fatalf("write json: %v", err)
	}
}

func TestApplyJSONFile_NoFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	cfg := Default()
	if err := applyJSONFile(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// defaults must be unchanged
	if cfg.Browser != "chromium" {
		t.Errorf("Browser=%q after missing file", cfg.Browser)
	}
}

func TestApplyJSONFile_OverridesFields(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	raw := map[string]any{
		"browser":            "firefox",
		"headless":           true,
		"verbose":            true,
		"timeout":            10000,
		"nav_timeout":        60000,
		"screenshot":         "always",
		"html_report":        true,
		"retries":            3,
		"verify_max_retries": 5,
		"workers":            4,
		"disable_cache":      true,
		"tags":               []string{"smoke", "ci"},
		"tests_home":         "mytests",
		"auto_annotate":      true,
		"break_lines":        []int{10, 20},
	}
	writeJSON(t, dir, raw)

	cfg := Default()
	if err := applyJSONFile(&cfg); err != nil {
		t.Fatalf("applyJSONFile: %v", err)
	}

	if cfg.Browser != "firefox" {
		t.Errorf("Browser=%q want firefox", cfg.Browser)
	}
	if !cfg.Headless {
		t.Error("Headless should be true")
	}
	if !cfg.Verbose {
		t.Error("Verbose should be true")
	}
	if cfg.DefaultTimeout != 10000*time.Millisecond {
		t.Errorf("DefaultTimeout=%v want 10s", cfg.DefaultTimeout)
	}
	if cfg.NavTimeout != 60000*time.Millisecond {
		t.Errorf("NavTimeout=%v want 60s", cfg.NavTimeout)
	}
	if cfg.Screenshot != "always" {
		t.Errorf("Screenshot=%q want always", cfg.Screenshot)
	}
	if !cfg.HTMLReport {
		t.Error("HTMLReport should be true")
	}
	if cfg.Retries != 3 {
		t.Errorf("Retries=%d want 3", cfg.Retries)
	}
	if cfg.VerifyMaxRetries != 5 {
		t.Errorf("VerifyMaxRetries=%d want 5", cfg.VerifyMaxRetries)
	}
	if cfg.Workers != 4 {
		t.Errorf("Workers=%d want 4", cfg.Workers)
	}
	if !cfg.DisableCache {
		t.Error("DisableCache should be true")
	}
	if len(cfg.Tags) != 2 || cfg.Tags[0] != "smoke" || cfg.Tags[1] != "ci" {
		t.Errorf("Tags=%v want [smoke ci]", cfg.Tags)
	}
	if cfg.TestsHome != "mytests" {
		t.Errorf("TestsHome=%q want mytests", cfg.TestsHome)
	}
	if !cfg.AutoAnnotate {
		t.Error("AutoAnnotate should be true")
	}
	if len(cfg.BreakLines) != 2 {
		t.Errorf("BreakLines=%v want [10,20]", cfg.BreakLines)
	}
}

func TestApplyJSONFile_DoesNotClobberDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	// Empty JSON object — nothing should change.
	writeJSON(t, dir, map[string]any{})

	cfg := Default()
	if err := applyJSONFile(&cfg); err != nil {
		t.Fatalf("applyJSONFile: %v", err)
	}
	if cfg.Browser != "chromium" {
		t.Errorf("Browser clobbered: got %q", cfg.Browser)
	}
	if cfg.DefaultTimeout != 5000*time.Millisecond {
		t.Errorf("DefaultTimeout clobbered: got %v", cfg.DefaultTimeout)
	}
	if cfg.Workers != 1 {
		t.Errorf("Workers clobbered: got %d", cfg.Workers)
	}
}

func TestApplyJSONFile_ExplicitFalseAndZero(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Start with a config where booleans are true and ints are non-zero,
	// then confirm JSON can explicitly set them to false/0.
	writeJSON(t, dir, map[string]any{
		"headless":           false,
		"verbose":            false,
		"html_report":        false,
		"disable_cache":      false,
		"auto_annotate":      false,
		"retries":            0,
		"verify_max_retries": 0,
		"workers":            0,
	})

	cfg := Default()
	// Override defaults to non-zero/true so we can confirm JSON sets them back.
	cfg.Headless = true
	cfg.Verbose = true
	cfg.HTMLReport = true
	cfg.DisableCache = true
	cfg.AutoAnnotate = true
	cfg.Retries = 5
	cfg.VerifyMaxRetries = 20
	cfg.Workers = 8

	if err := applyJSONFile(&cfg); err != nil {
		t.Fatalf("applyJSONFile: %v", err)
	}

	if cfg.Headless {
		t.Error("Headless: JSON false should override true")
	}
	if cfg.Verbose {
		t.Error("Verbose: JSON false should override true")
	}
	if cfg.HTMLReport {
		t.Error("HTMLReport: JSON false should override true")
	}
	if cfg.DisableCache {
		t.Error("DisableCache: JSON false should override true")
	}
	if cfg.AutoAnnotate {
		t.Error("AutoAnnotate: JSON false should override true")
	}
	if cfg.Retries != 0 {
		t.Errorf("Retries=%d: JSON 0 should override 5", cfg.Retries)
	}
	if cfg.VerifyMaxRetries != 0 {
		t.Errorf("VerifyMaxRetries=%d: JSON 0 should override 20", cfg.VerifyMaxRetries)
	}
	if cfg.Workers != 0 {
		t.Errorf("Workers=%d: JSON 0 should override 8", cfg.Workers)
	}
}

func TestApplyJSONFile_BrokenJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(dir, "manul_engine_configuration.json"), []byte("not json"), 0644)

	cfg := Default()
	err := applyJSONFile(&cfg)
	if err == nil {
		t.Fatal("expected error for broken JSON")
	}
}

func TestApplyJSONFile_ExecutablePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	writeJSON(t, dir, map[string]any{"executable_path": "/usr/bin/chromium"})

	cfg := Default()
	if err := applyJSONFile(&cfg); err != nil {
		t.Fatalf("applyJSONFile: %v", err)
	}
	if cfg.ExecutablePath == nil || *cfg.ExecutablePath != "/usr/bin/chromium" {
		t.Fatal("ExecutablePath should be set from JSON")
	}
}

// ---- overrideFromEnv --------------------------------------------------------

func TestOverrideFromEnv(t *testing.T) {
	cases := []struct {
		name    string
		envKey  string
		envVal  string
		check   func(Config) bool
		wantMsg string
	}{
		{
			"CDP endpoint",
			"MANUL_CDP_ENDPOINT", "http://127.0.0.1:9222",
			func(c Config) bool { return c.CDPEndpoint == "http://127.0.0.1:9222" },
			"CDPEndpoint mismatch",
		},
		{
			"browser",
			"MANUL_BROWSER", "webkit",
			func(c Config) bool { return c.Browser == "webkit" },
			"Browser mismatch",
		},
		{
			"headless true",
			"MANUL_HEADLESS", "true",
			func(c Config) bool { return c.Headless },
			"Headless should be true",
		},
		{
			"headless false",
			"MANUL_HEADLESS", "false",
			func(c Config) bool { return !c.Headless },
			"Headless should be false",
		},
		{
			"verbose",
			"MANUL_VERBOSE", "1",
			func(c Config) bool { return c.Verbose },
			"Verbose should be true",
		},
		{
			"debug mode",
			"MANUL_DEBUG", "true",
			func(c Config) bool { return c.DebugMode },
			"DebugMode should be true",
		},
		{
			"explain mode",
			"MANUL_EXPLAIN", "true",
			func(c Config) bool { return c.ExplainMode },
			"ExplainMode should be true",
		},
		{
			"timeout int ms",
			"MANUL_TIMEOUT", "8000",
			func(c Config) bool { return c.DefaultTimeout == 8000*time.Millisecond },
			"DefaultTimeout mismatch",
		},
		{
			"timeout duration string",
			"MANUL_TIMEOUT", "3s",
			func(c Config) bool { return c.DefaultTimeout == 3*time.Second },
			"DefaultTimeout from duration string",
		},
		{
			"nav timeout",
			"MANUL_NAV_TIMEOUT", "45000",
			func(c Config) bool { return c.NavTimeout == 45000*time.Millisecond },
			"NavTimeout mismatch",
		},
		{
			"screenshot",
			"MANUL_SCREENSHOT", "always",
			func(c Config) bool { return c.Screenshot == "always" },
			"Screenshot mismatch",
		},
		{
			"html report",
			"MANUL_HTML_REPORT", "true",
			func(c Config) bool { return c.HTMLReport },
			"HTMLReport should be true",
		},
		{
			"retries",
			"MANUL_RETRIES", "5",
			func(c Config) bool { return c.Retries == 5 },
			"Retries mismatch",
		},
		{
			"verify max retries",
			"MANUL_VERIFY_MAX_RETRIES", "20",
			func(c Config) bool { return c.VerifyMaxRetries == 20 },
			"VerifyMaxRetries mismatch",
		},
		{
			"workers",
			"MANUL_WORKERS", "8",
			func(c Config) bool { return c.Workers == 8 },
			"Workers mismatch",
		},
		{
			"disable cache",
			"MANUL_DISABLE_CACHE", "true",
			func(c Config) bool { return c.DisableCache },
			"DisableCache should be true",
		},
		{
			"semantic cache disabled (ManulEngine parity)",
			"MANUL_SEMANTIC_CACHE_ENABLED", "false",
			func(c Config) bool { return c.DisableCache },
			"MANUL_SEMANTIC_CACHE_ENABLED=false should set DisableCache=true",
		},
		{
			"tags",
			"MANUL_TAGS", "smoke,ci,regression",
			func(c Config) bool {
				return len(c.Tags) == 3 && c.Tags[0] == "smoke" && c.Tags[1] == "ci" && c.Tags[2] == "regression"
			},
			"Tags mismatch",
		},
		{
			"tests home",
			"MANUL_TESTS_HOME", "/opt/tests",
			func(c Config) bool { return c.TestsHome == "/opt/tests" },
			"TestsHome mismatch",
		},
		{
			"channel (ManulEngine parity)",
			"MANUL_CHANNEL", "chrome",
			func(c Config) bool { return c.Channel != nil && *c.Channel == "chrome" },
			"MANUL_CHANNEL should set Channel",
		},
		{
			"auto annotate",
			"MANUL_AUTO_ANNOTATE", "true",
			func(c Config) bool { return c.AutoAnnotate },
			"AutoAnnotate should be true",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.envKey, tc.envVal)
			cfg := Default()
			overrideFromEnv(&cfg)
			if !tc.check(cfg) {
				t.Errorf("%s (env %s=%s)", tc.wantMsg, tc.envKey, tc.envVal)
			}
		})
	}
}

func TestOverrideFromEnv_BrowserArgs(t *testing.T) {
	t.Setenv("MANUL_BROWSER_ARGS", "--no-sandbox, --disable-gpu")
	cfg := Default()
	overrideFromEnv(&cfg)
	if len(cfg.BrowserArgs) != 2 {
		t.Fatalf("BrowserArgs=%v want 2 elements", cfg.BrowserArgs)
	}
	if cfg.BrowserArgs[0] != "--no-sandbox" {
		t.Errorf("BrowserArgs[0]=%q", cfg.BrowserArgs[0])
	}
	if cfg.BrowserArgs[1] != "--disable-gpu" {
		t.Errorf("BrowserArgs[1]=%q", cfg.BrowserArgs[1])
	}
}

func TestOverrideFromEnv_ExecutablePath(t *testing.T) {
	t.Setenv("MANUL_EXECUTABLE_PATH", "/usr/bin/chromium")
	cfg := Default()
	overrideFromEnv(&cfg)
	if cfg.ExecutablePath == nil {
		t.Fatal("ExecutablePath should not be nil")
	}
	if *cfg.ExecutablePath != "/usr/bin/chromium" {
		t.Errorf("ExecutablePath=%q want /usr/bin/chromium", *cfg.ExecutablePath)
	}
}

// ---- Load() priority chain --------------------------------------------------

func TestLoad_EnvBeatsJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	writeJSON(t, dir, map[string]any{"browser": "firefox"})
	t.Setenv("MANUL_BROWSER", "webkit")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Browser != "webkit" {
		t.Errorf("Browser=%q want webkit (env beats JSON)", cfg.Browser)
	}
}

func TestLoad_JSONBeatsDefault(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	writeJSON(t, dir, map[string]any{"browser": "firefox"})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Browser != "firefox" {
		t.Errorf("Browser=%q want firefox (JSON beats default)", cfg.Browser)
	}
}

func TestLoad_NoFileNoEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	def := Default()
	if cfg.Browser != def.Browser {
		t.Errorf("Browser=%q want default %q", cfg.Browser, def.Browser)
	}
	if cfg.Workers != def.Workers {
		t.Errorf("Workers=%d want default %d", cfg.Workers, def.Workers)
	}
}

func TestLoad_BrokenJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(dir, "manul_engine_configuration.json"), []byte("not json"), 0644)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for broken JSON")
	}
}

func TestLoad_EnvEmptyStringDoesNotOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	writeJSON(t, dir, map[string]any{"browser": "firefox"})
	t.Setenv("MANUL_BROWSER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Browser != "firefox" {
		t.Errorf("Browser=%q want firefox (empty env should not override JSON)", cfg.Browser)
	}
}
