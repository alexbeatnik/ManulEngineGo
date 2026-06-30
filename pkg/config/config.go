// Package config holds the engine-wide runtime configuration for ManulEngine (Go).
// Each hunt execution gets a Config passed through the runtime stack.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config is the engine-wide runtime configuration.
// Populated via Load(), which applies env vars > JSON file > hardcoded defaults.
type Config struct {
	// CDPEndpoint is the Chrome DevTools Protocol HTTP endpoint.
	// Example: "http://127.0.0.1:9222"
	CDPEndpoint string `json:"cdp_endpoint"`

	// Browser is the browser type to launch ("chromium", "firefox", "webkit").
	Browser string `json:"browser"`

	// BrowserArgs are extra flags passed to the browser process on launch.
	BrowserArgs []string `json:"browser_args"`

	// ExecutablePath overrides the browser binary location when non-nil.
	ExecutablePath *string `json:"executable_path,omitempty"`

	// Channel selects a system Chrome/Chromium binary by name (chrome,
	// chrome-beta, chrome-dev, chromium, msedge). Mirrors ManulEngine's channel.
	Channel *string `json:"channel,omitempty"`

	// Headless runs the browser in headless mode.
	Headless bool `json:"headless"`

	// Verbose enables verbose logging.
	Verbose bool `json:"verbose"`

	// DebugMode pauses execution between each DSL command for interactive stepping.
	DebugMode bool `json:"debug_mode"`

	// BreakLines is a list of .hunt file line numbers (1-based) that act as
	// breakpoints when DebugMode is true. Empty means pause on every step.
	BreakLines []int `json:"break_lines"`

	// ExplainMode prints a full scoring breakdown for every targeted element.
	ExplainMode bool `json:"explain_mode"`

	// DefaultTimeout is the per-command context deadline.
	DefaultTimeout time.Duration `json:"timeout"`

	// NavTimeout is the maximum time allowed for page navigation.
	NavTimeout time.Duration `json:"nav_timeout"`

	// Screenshot controls when screenshots are taken: "none", "on-fail", "always".
	Screenshot string `json:"screenshot"`

	// HTMLReport enables generation of an HTML execution report.
	HTMLReport bool `json:"html_report"`

	// Retries is the number of times to retry a failed targeting command.
	Retries int `json:"retries"`

	// VerifyMaxRetries is the poll limit for VERIFY assertions before failing.
	VerifyMaxRetries int `json:"verify_max_retries"`

	// Workers is the number of parallel hunt workers in pool mode.
	Workers int `json:"workers"`

	// DisableCache disables the DOM snapshot cache (forces re-probe on every command).
	DisableCache bool `json:"disable_cache"`

	// Tags filters which @tag-annotated commands to execute.
	// Empty slice means run all commands.
	Tags []string `json:"tags"`

	// TestsHome is the root directory where hunt files are resolved.
	TestsHome string `json:"tests_home"`

	// AutoAnnotate automatically adds @tag annotations based on heuristic signals.
	AutoAnnotate bool `json:"auto_annotate"`
}

// jsonConfig is an intermediate struct for JSON unmarshaling.
// All booleans and ints use pointer types so that explicit false/0 in JSON
// is distinguishable from a missing field (nil pointer = absent).
// timeout and nav_timeout are stored as integer milliseconds in JSON.
type jsonConfig struct {
	CDPEndpoint      string   `json:"cdp_endpoint"`
	Browser          string   `json:"browser"`
	BrowserArgs      []string `json:"browser_args"`
	ExecutablePath   *string  `json:"executable_path"`
	Channel          *string  `json:"channel"`
	Headless         *bool    `json:"headless"`
	Verbose          *bool    `json:"verbose"`
	DebugMode        *bool    `json:"debug_mode"`
	BreakLines       []int    `json:"break_lines"`
	ExplainMode      *bool    `json:"explain_mode"`
	TimeoutMs        *int     `json:"timeout"`
	NavTimeoutMs     *int     `json:"nav_timeout"`
	Screenshot       string   `json:"screenshot"`
	HTMLReport       *bool    `json:"html_report"`
	Retries          *int     `json:"retries"`
	VerifyMaxRetries *int     `json:"verify_max_retries"`
	Workers          *int     `json:"workers"`
	DisableCache     *bool    `json:"disable_cache"`
	Tags             []string `json:"tags"`
	TestsHome        string   `json:"tests_home"`
	AutoAnnotate     *bool    `json:"auto_annotate"`
}

// Default returns a Config with production defaults matching the Python CLI contract.
func Default() Config {
	return Config{
		Browser:          "chromium",
		Headless:         false,
		DefaultTimeout:   5000 * time.Millisecond,
		NavTimeout:       30000 * time.Millisecond,
		Screenshot:       "on-fail",
		HTMLReport:       false,
		Retries:          0,
		VerifyMaxRetries: 15,
		Workers:          1,
		TestsHome:        "tests",
	}
}

// Load returns a Config with the 3-level priority applied:
//
//  1. Hardcoded defaults (Default())
//  2. manul_engine_configuration.json in CWD (if present)
//  3. MANUL_* environment variables (highest priority)
func Load() (Config, error) {
	cfg := Default()

	if err := applyJSONFile(&cfg); err != nil {
		return cfg, err
	}
	overrideFromEnv(&cfg)
	return cfg, nil
}

// applyJSONFile reads manul_engine_configuration.json from the CWD and overlays
// any fields present in the file onto cfg.
func applyJSONFile(cfg *Config) error {
	cwd, err := os.Getwd()
	if err != nil {
		return nil // non-fatal; proceed with defaults
	}
	path := filepath.Join(cwd, "manul_engine_configuration.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var raw jsonConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.CDPEndpoint != "" {
		cfg.CDPEndpoint = raw.CDPEndpoint
	}
	if raw.Browser != "" {
		cfg.Browser = raw.Browser
	}
	if raw.BrowserArgs != nil {
		cfg.BrowserArgs = raw.BrowserArgs
	}
	if raw.ExecutablePath != nil {
		cfg.ExecutablePath = raw.ExecutablePath
	}
	if raw.Channel != nil {
		cfg.Channel = raw.Channel
	}
	if raw.Headless != nil {
		cfg.Headless = *raw.Headless
	}
	if raw.Verbose != nil {
		cfg.Verbose = *raw.Verbose
	}
	if raw.DebugMode != nil {
		cfg.DebugMode = *raw.DebugMode
	}
	if len(raw.BreakLines) > 0 {
		cfg.BreakLines = raw.BreakLines
	}
	if raw.ExplainMode != nil {
		cfg.ExplainMode = *raw.ExplainMode
	}
	if raw.TimeoutMs != nil {
		cfg.DefaultTimeout = time.Duration(*raw.TimeoutMs) * time.Millisecond
	}
	if raw.NavTimeoutMs != nil {
		cfg.NavTimeout = time.Duration(*raw.NavTimeoutMs) * time.Millisecond
	}
	if raw.Screenshot != "" {
		cfg.Screenshot = raw.Screenshot
	}
	if raw.HTMLReport != nil {
		cfg.HTMLReport = *raw.HTMLReport
	}
	if raw.Retries != nil {
		cfg.Retries = *raw.Retries
	}
	if raw.VerifyMaxRetries != nil {
		cfg.VerifyMaxRetries = *raw.VerifyMaxRetries
	}
	if raw.Workers != nil {
		cfg.Workers = *raw.Workers
	}
	if raw.DisableCache != nil {
		cfg.DisableCache = *raw.DisableCache
	}
	if len(raw.Tags) > 0 {
		cfg.Tags = raw.Tags
	}
	if raw.TestsHome != "" {
		cfg.TestsHome = raw.TestsHome
	}
	if raw.AutoAnnotate != nil {
		cfg.AutoAnnotate = *raw.AutoAnnotate
	}
	return nil
}

// overrideFromEnv applies MANUL_* environment variables onto cfg.
// Only non-empty env values override; this preserves JSON/default values.
func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("MANUL_CDP_ENDPOINT"); v != "" {
		cfg.CDPEndpoint = v
	}
	if v := os.Getenv("MANUL_BROWSER"); v != "" {
		cfg.Browser = v
	}
	if v := os.Getenv("MANUL_BROWSER_ARGS"); v != "" {
		cfg.BrowserArgs = splitCSV(v)
	}
	if v := os.Getenv("MANUL_EXECUTABLE_PATH"); v != "" {
		cfg.ExecutablePath = &v
	}
	if v := os.Getenv("MANUL_CHANNEL"); v != "" {
		cfg.Channel = &v
	}
	if v := os.Getenv("MANUL_HEADLESS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Headless = b
		}
	}
	if v := os.Getenv("MANUL_VERBOSE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Verbose = b
		}
	}
	if v := os.Getenv("MANUL_DEBUG"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.DebugMode = b
		}
	}
	if v := os.Getenv("MANUL_EXPLAIN"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.ExplainMode = b
		}
	}
	if v := os.Getenv("MANUL_TIMEOUT"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil {
			cfg.DefaultTimeout = time.Duration(ms) * time.Millisecond
		} else if d, err := time.ParseDuration(v); err == nil {
			cfg.DefaultTimeout = d
		}
	}
	if v := os.Getenv("MANUL_NAV_TIMEOUT"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil {
			cfg.NavTimeout = time.Duration(ms) * time.Millisecond
		} else if d, err := time.ParseDuration(v); err == nil {
			cfg.NavTimeout = d
		}
	}
	if v := os.Getenv("MANUL_SCREENSHOT"); v != "" {
		cfg.Screenshot = v
	}
	if v := os.Getenv("MANUL_HTML_REPORT"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.HTMLReport = b
		}
	}
	if v := os.Getenv("MANUL_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Retries = n
		}
	}
	if v := os.Getenv("MANUL_VERIFY_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.VerifyMaxRetries = n
		}
	}
	if v := os.Getenv("MANUL_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Workers = n
		}
	}
	if v := os.Getenv("MANUL_DISABLE_CACHE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.DisableCache = b
		}
	}
	if v := os.Getenv("MANUL_SEMANTIC_CACHE_ENABLED"); v != "" {
		// ManulEngine parity: the inverse control — disabling the cache is
		// equivalent to DisableCache=true. Read after MANUL_DISABLE_CACHE so
		// the explicit semantic-cache toggle wins if both are set.
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.DisableCache = !b
		}
	}
	if v := os.Getenv("MANUL_TAGS"); v != "" {
		cfg.Tags = splitCSV(v)
	}
	if v := os.Getenv("MANUL_TESTS_HOME"); v != "" {
		cfg.TestsHome = v
	}
	if v := os.Getenv("MANUL_AUTO_ANNOTATE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.AutoAnnotate = b
		}
	}
}

// splitCSV splits a comma-separated string, trimming whitespace from each part.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
