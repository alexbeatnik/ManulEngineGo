// ManulHeart driver — CLI entry point.
//
// Usage:
//
//	manul <hunt-file>                 run a single hunt file
//	manul <directory>                 run all .hunt files in the directory
//	manul .                           run all .hunt files in the current directory
//	manul run <hunt-file> [flags]     explicit run subcommand (same as above)
//	manul run-step '<DSL command>'    execute a single DSL command
//
// Examples:
//
//	manul examples/saucedemo.hunt
//	manul examples/saucedemo.hunt --headless
//	manul examples/
//	manul .
//	manul run examples/saucedemo.hunt --cdp http://127.0.0.1:9222
//	manul run-step "Click the 'Login' button" --cdp http://127.0.0.1:9222
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexbeatnik/ManulHeart/pkg/agent"
	"github.com/alexbeatnik/ManulHeart/pkg/browser"
	"github.com/alexbeatnik/ManulHeart/pkg/config"
	"github.com/alexbeatnik/ManulHeart/pkg/daemon"
	"github.com/alexbeatnik/ManulHeart/pkg/data"
	"github.com/alexbeatnik/ManulHeart/pkg/dsl"
	"github.com/alexbeatnik/ManulHeart/pkg/explain"
	"github.com/alexbeatnik/ManulHeart/pkg/pages"
	"github.com/alexbeatnik/ManulHeart/pkg/record"
	"github.com/alexbeatnik/ManulHeart/pkg/report"
	"github.com/alexbeatnik/ManulHeart/pkg/runtime"
	"github.com/alexbeatnik/ManulHeart/pkg/scan"
	"github.com/alexbeatnik/ManulHeart/pkg/utils"
	"github.com/alexbeatnik/ManulHeart/pkg/worker"
)

// version is the single source of truth for the engine version. It tracks the
// git module tag (semver vX.Y.Z) so `manul --version`, the README, and
// `go get ...@<tag>` all agree. Bump this together with the tag.
const version = "v0.0.9"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	firstArg := os.Args[1]

	if firstArg == "--version" || firstArg == "-version" || firstArg == "-v" {
		fmt.Printf("manul %s\n", version)
		os.Stdout.Sync()
		return
	}

	// If the first arg is a flag (starts with -), or it's a known target-like string,
	// treat the whole execution as an implicit "run".
	if strings.HasPrefix(firstArg, "-") || looksLikeTarget(firstArg) {
		if err := cmdRun(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	switch firstArg {
	case "run":
		if err := cmdRun(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "run-step":
		if err := cmdRunStep(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "read":
		if err := cmdRead(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "map":
		if err := cmdMap(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "schema":
		if err := cmdSchema(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "scan":
		if err := cmdScan(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "record":
		if err := cmdRecord(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "daemon":
		if err := cmdDaemon(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "pages":
		if err := cmdPages(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "controls":
		if err := cmdControls(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %q\n\n", firstArg)
		printUsage()
		os.Exit(1)
	}
}

// looksLikeTarget returns true if the argument appears to be a .hunt file,
// a directory path, or "." — i.e. something to run directly.
func looksLikeTarget(arg string) bool {
	if arg == "." {
		return true
	}
	if strings.HasSuffix(arg, ".hunt") {
		return true
	}
	// Check if it's an existing directory.
	info, err := os.Stat(arg)
	if err == nil && info.IsDir() {
		return true
	}
	return false
}

// ── run subcommand ────────────────────────────────────────────────────────────

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	cdpEndpoint := fs.String("cdp", "", "CDP endpoint URL (if set, skip auto-launch and connect to existing browser)")
	verbose := fs.Bool("verbose", false, "enable verbose logging")
	jsonOut := fs.Bool("json", false, "print JSON result to stdout")
	jsonlOut := fs.Bool("jsonl", false, "stream per-step JSON Lines + final HuntResult to stdout (implies -json semantics)")
	targetSelector := fs.String("target", "", "CDP tab selector, e.g. 'url=youtube.com' to pick the most recently active tab whose URL contains 'youtube.com'")
	timeout := fs.Duration("timeout", 30*time.Second, "default command timeout")
	userDataDir := fs.String("user-data-dir", "", "Chrome profile directory (empty = unique temp dir per run)")
	headless := fs.Bool("headless", false, "run Chrome in headless mode")
	debug := fs.Bool("debug", false, "enable debug mode (pause on each step)")
	explainMode := fs.Bool("explain", false, "enable explain mode (show targeting candidates)")
	screenshot := fs.String("screenshot", "on-fail", "screenshot mode: none, on-fail, always")
	htmlReport := fs.Bool("html-report", true, "generate HTML report after run")
	executablePath := fs.String("executable-path", "", "absolute path to a custom browser or Electron app executable")
	tags := fs.String("tags", "", "comma-separated tags to filter hunt files")
	retries := fs.Int("retries", 0, "number of retries for failed steps")
	disableCache := fs.Bool("disable-cache", false, "disable DOM snapshot caching")
	_ = fs.Int("workers", 1, "number of parallel workers (placeholder for compatibility)")
	_ = fs.String("browser", "chromium", "browser type (default: chromium)")
	breakLinesStr := fs.String("break-lines", "", "comma-separated line numbers to pause on (debugging)")
	showVersion := fs.Bool("version", false, "show engine version and exit")

	var target string
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: manul <hunt-file|directory|-> [flags]\n\n  Use '-' as target to read the hunt script from stdin.\n\nFlags:\n")
		fs.PrintDefaults()
	}

	remaining := args
	for len(remaining) > 0 {
		if err := fs.Parse(remaining); err != nil {
			return err
		}
		if fs.NArg() > 0 {
			if target == "" {
				target = fs.Arg(0)
			}
			remaining = fs.Args()[1:]
		} else {
			remaining = nil
		}
	}

	if *showVersion {
		fmt.Printf("manul %s\n", version)
		os.Stdout.Sync()
		return nil
	}
 
	if target == "" {
		fs.Usage()
		return fmt.Errorf("hunt file or directory path is required")
	}

	// stdinHunt is populated when target is "-": read the .hunt script from
	// stdin instead of a file. Imports resolve relative to the current
	// working directory in that mode (no source path to anchor against).
	var stdinHunt *dsl.Hunt
	var huntFiles []string
	if target == "-" {
		h, perr := dsl.Parse(os.Stdin)
		if perr != nil {
			return fmt.Errorf("parse stdin: %w", perr)
		}
		h.SourcePath = "<stdin>"
		stdinHunt = h
	} else {
		// Collect .hunt files from target.
		var err error
		huntFiles, err = collectHuntFiles(target)
		if err != nil {
			return err
		}
		if len(huntFiles) == 0 {
			return fmt.Errorf("no .hunt files found in %q", target)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// CLI --cdp wins over JSON/env. Without this override, runSequential
	// receives the right cdpEndpoint argument but cfg.CDPEndpoint stays
	// at config.Default() (""), and the per-hunt CDP banner / NewCDPBrowser
	// call see an empty endpoint — which downstream resolves to the bogus
	// "http://json/list" URL.
	if *cdpEndpoint != "" {
		cfg.CDPEndpoint = *cdpEndpoint
	}

	cfg.Verbose = *verbose
	if *timeout != 30*time.Second { // only override if user provided a flag
		cfg.DefaultTimeout = *timeout
	}
	cfg.DebugMode = *debug
	if *explainMode {
		cfg.ExplainMode = true
	}
	if *screenshot != "none" {
		cfg.Screenshot = *screenshot
	}
	cfg.HTMLReport = *htmlReport
	cfg.Retries = *retries
	cfg.DisableCache = *disableCache
	if *tags != "" {
		cfg.Tags = strings.Split(*tags, ",")
		for i := range cfg.Tags {
			cfg.Tags[i] = strings.TrimSpace(cfg.Tags[i])
		}
	}
	if *breakLinesStr != "" {
		cfg.DebugMode = true
		for _, part := range strings.Split(*breakLinesStr, ",") {
			if ln, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
				cfg.BreakLines = append(cfg.BreakLines, ln)
			}
		}
	}

	logLevel := utils.LogLevelInfo
	if *verbose {
		logLevel = utils.LogLevelDebug
	}
	// When emitting JSON, keep stdout exclusively for the structured
	// payload — route human-readable logs to stderr so callers can
	// `json.Unmarshal(stdout)` directly. -jsonl shares the same routing
	// rule: stdout is reserved for the streamed payload + final summary.
	jsonStdout := *jsonOut || *jsonlOut
	var logger *utils.Logger
	if jsonStdout {
		logger = utils.NewLoggerTo(os.Stderr, nil).WithLevel(logLevel)
	} else {
		logger = utils.NewLogger(nil).WithLevel(logLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var hunts []*dsl.Hunt
	parseFailed := 0

	if stdinHunt != nil {
		fmt.Fprintf(os.Stderr, "🐾 Manul: reading hunt from stdin\n")
		if err := dsl.ResolveImports(stdinHunt); err != nil {
			return fmt.Errorf("imports: %w", err)
		}
		if err := stdinHunt.Expand(); err != nil {
			return fmt.Errorf("expand: %w", err)
		}
		hunts = []*dsl.Hunt{stdinHunt}
	} else {
		fmt.Fprintf(os.Stderr, "🐾 Manul: found %d hunt file(s)\n", len(huntFiles))
		// Pre-parse all hunts so we can decide between sequential and parallel modes.
		for _, huntFile := range huntFiles {
			hunt, err := dsl.ParseFile(huntFile)
			if err != nil {
				logger.Error("parse %q: %v", huntFile, err)
				parseFailed++
				continue
			}
			if err := dsl.ResolveImports(hunt); err != nil {
				logger.Error("imports %q: %v", huntFile, err)
				parseFailed++
				continue
			}
			if err := hunt.Expand(); err != nil {
				logger.Error("expand %q: %v", huntFile, err)
				parseFailed++
				continue
			}
			hunts = append(hunts, hunt)
		}
	}

	if len(hunts) == 0 {
		return fmt.Errorf("no hunt files could be parsed")
	}

	opts := outputOpts{json: *jsonOut, jsonl: *jsonlOut}
	tabURLSubstr, terr := parseTargetSelector(*targetSelector)
	if terr != nil {
		return terr
	}

	var totalFailed int
	if cfg.Workers > 1 && len(hunts) > 1 {
		// Parallel execution via worker pool.
		if tabURLSubstr != "" {
			logger.Warn("-target is ignored in --workers >1 (each worker spawns its own Chrome)")
		}
		totalFailed = runParallel(ctx, cfg, hunts, opts, logger)
	} else {
		// Sequential execution.
		totalFailed = runSequential(ctx, cfg, hunts, opts, *cdpEndpoint, *userDataDir, *headless, *executablePath, tabURLSubstr, logger)
	}

	totalFailed += parseFailed
	if totalFailed > 0 {
		// Use hunts (not huntFiles) so stdin runs report "1/1" instead
		// of "1/0".
		return fmt.Errorf("%d/%d hunt file(s) failed", totalFailed, len(hunts))
	}
	return nil
}

// outputOpts decides how the CLI emits results to stdout. -json prints a
// single HuntResult at the end; -jsonl streams per-step JSON Lines plus
// a final HuntResult line. Either flag routes the logger to stderr so
// stdout stays clean for the structured payload.
type outputOpts struct {
	json  bool // print final HuntResult as indented JSON
	jsonl bool // stream per-step ExecutionResult lines + final HuntResult
}

func (o outputOpts) anyJSON() bool { return o.json || o.jsonl }

// parseTargetSelector decodes the -target flag value. Currently only
// "url=<substring>" is recognized — it asks the CDP backend to pick the
// most-recently-active page whose URL contains <substring>. Empty input
// is valid and means "fall back to first-page semantics".
func parseTargetSelector(raw string) (urlSubstr string, err error) {
	if raw == "" {
		return "", nil
	}
	const prefix = "url="
	if !strings.HasPrefix(raw, prefix) {
		return "", fmt.Errorf("--target: unsupported selector %q (expected 'url=<substring>')", raw)
	}
	v := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	if v == "" {
		return "", fmt.Errorf("--target: empty 'url=' value")
	}
	return v, nil
}

// newRuntimeWithStreaming wires runtime.New with an onStep callback when
// -jsonl is active. Each step's ExecutionResult is encoded as a single
// compact JSON line tagged with `"event":"step"` so consumers can tell
// streaming rows apart from the final HuntResult.
func newRuntimeWithStreaming(cfg config.Config, page browser.Page, logger *utils.Logger, opts outputOpts) *runtime.Runtime {
	rt := runtime.New(cfg, page, logger)
	if opts.jsonl {
		enc := json.NewEncoder(os.Stdout)
		var streamMu sync.Mutex // workers may share stdout
		rt.SetOnStep(func(r explain.ExecutionResult) {
			streamMu.Lock()
			defer streamMu.Unlock()
			// Wrap in an envelope so the line is self-describing.
			_ = enc.Encode(map[string]any{
				"event": "step",
				"data":  r,
			})
			os.Stdout.Sync()
		})
	}
	return rt
}

// runSequential executes hunts one at a time, launching a single Chrome when needed.
//
// tabURLSubstr, when non-empty, asks the CDP backend to attach to the
// page whose URL contains this substring (case-insensitive). Empty means
// "first available page", which matches Chrome's most-recently-active
// tab in practice.
func runSequential(ctx context.Context, cfg config.Config, hunts []*dsl.Hunt, opts outputOpts, cdpEndpoint, userDataDir string, headless bool, executablePath, tabURLSubstr string, logger *utils.Logger) int {
	var chrome *browser.ChromeProcess
	if cdpEndpoint == "" {
		opts := browser.DefaultChromeOptions()
		opts.UserDataDir = userDataDir
		if headless {
			cfg.Headless = true
		}
		opts.Headless = cfg.Headless
		if executablePath != "" {
			opts.ExecutablePath = executablePath
		}
		logger.Info("Launching Chrome (port %d, profile %s)…", opts.Port, opts.UserDataDir)
		var err error
		chrome, err = browser.LaunchChrome(ctx, opts)
		if err != nil {
			logger.Error("launch chrome: %v", err)
			return len(hunts)
		}
		var closeOnce sync.Once
		closeChrome := func() {
			closeOnce.Do(func() {
				logger.Debug("Closing Chrome…")
				chrome.Close()
			})
		}
		defer closeChrome()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			logger.Debug("Signal received, cancelling…")
			closeChrome()
		}()
		cfg.CDPEndpoint = chrome.Endpoint()
	}

	var totalFailed int
	for i, hunt := range hunts {
		filename := filepath.Base(hunt.SourcePath)
		if len(hunts) > 1 {
			fmt.Fprintf(os.Stderr, "\n%s\n📜 [%d/%d] %s\n%s\n",
				strings.Repeat("=", 60), i+1, len(hunts), filename, strings.Repeat("=", 60))
		} else {
			fmt.Fprintf(os.Stderr, "\n%s\n📜 %s\n%s\n",
				strings.Repeat("=", 60), filename, strings.Repeat("=", 60))
		}

		logger.Info("ManulHeart — %s", hunt.SourcePath)
		if hunt.Title != "" {
			logger.Info("Title: %s", hunt.Title)
		}
		logger.Info("Commands: %d", len(hunt.Commands))
		logger.Info("CDP: %s", cfg.CDPEndpoint)

		b := browser.NewCDPBrowser(cfg.CDPEndpoint)
		var page browser.Page
		var err error
		if tabURLSubstr != "" {
			page, err = b.PageMatching(ctx, tabURLSubstr)
		} else {
			page, err = b.FirstPage(ctx)
		}
		if err != nil {
			logger.Error("connect to browser at %q: %v", cfg.CDPEndpoint, err)
			totalFailed++
			continue
		}
		func() {
			defer page.Close()

			// Data-driven testing: if @data: is declared, load rows and run once per row.
			if hunt.DataFile != "" {
				rows, dErr := data.LoadFile(hunt.DataFile, filepath.Dir(hunt.SourcePath))
				if dErr != nil {
					logger.Error("load data file %q: %v", hunt.DataFile, dErr)
					totalFailed++
					return
				}
				if len(rows) == 0 {
					logger.Warn("data file %q is empty — running once with no extra vars", hunt.DataFile)
				} else {
					logger.Info("📊 Data-Driven: %d rows loaded from %q", len(rows), hunt.DataFile)
				}

				allOk := true
				for rowIdx, row := range rows {
					if len(rows) > 1 {
						fmt.Fprintf(os.Stderr, "\n%s\n📊 Data row %d/%d: %v\n%s\n",
							strings.Repeat("-", 40), rowIdx+1, len(rows), row, strings.Repeat("-", 40))
					}
					rt := newRuntimeWithStreaming(cfg, page, logger, opts)
					result, runErr := rt.RunHunt(ctx, hunt, row)
					if runErr != nil {
						logger.Error("hunt %q row %d failed: %v", hunt.SourcePath, rowIdx+1, runErr)
						allOk = false
					}
					printResult(result, opts, logger)
					if hErr := report.AppendRunHistory("reports", result); hErr != nil {
						logger.Warn("run_history append failed: %v", hErr)
					}
					if cfg.HTMLReport {
						reportPath, rErr := report.GenerateHTML(result, "reports")
						if rErr != nil {
							logger.Warn("HTML report generation failed: %v", rErr)
						} else {
							logger.Info("📊 HTML report: %s", reportPath)
						}
					}
					if result != nil && !result.Success {
						allOk = false
					}
				}
				if !allOk {
					totalFailed++
				}
				return
			}

			// Standard (non-data-driven) execution.
			rt := newRuntimeWithStreaming(cfg, page, logger, opts)
			result, err := rt.RunHunt(ctx, hunt)
			if err != nil {
				logger.Error("hunt %q failed: %v", hunt.SourcePath, err)
				totalFailed++
				// RunHunt returns a partial *HuntResult even on failure;
				// emit it so downstream consumers (e.g. the OS-Manul
				// dispatcher) can read per-step errors instead of being
				// limited to "exit 1".
				if result != nil {
					printResult(result, opts, logger)
				}
				return
			}
			printResult(result, opts, logger)
			if hErr := report.AppendRunHistory("reports", result); hErr != nil {
				logger.Warn("run_history append failed: %v", hErr)
			}
			if cfg.HTMLReport {
				reportPath, rErr := report.GenerateHTML(result, "reports")
				if rErr != nil {
					logger.Warn("HTML report generation failed: %v", rErr)
				} else {
					logger.Info("📊 HTML report: %s", reportPath)
				}
			}
			if !result.Success {
				totalFailed++
			}
		}()
	}
	return totalFailed
}

// runParallel executes hunts across a worker pool.
func runParallel(ctx context.Context, cfg config.Config, hunts []*dsl.Hunt, opts outputOpts, logger *utils.Logger) int {
	if opts.jsonl {
		// pkg/worker spins its own runtime.Runtime per hunt; per-step
		// streaming would need a worker-level callback we haven't
		// surfaced yet. Degrade gracefully to whole-hunt JSON envelopes.
		fmt.Fprintln(os.Stderr, "warning: -jsonl per-step streaming is not yet supported with --workers >1; emitting hunt-level events only")
	}
	fmt.Fprintf(os.Stderr, "🐾 Running %d hunts in parallel (workers: %d)\n", len(hunts), cfg.Workers)
	results, err := worker.RunHuntsInParallel(ctx, cfg, hunts, cfg.Workers, logger)
	if err != nil {
		logger.Error("parallel run failed: %v", err)
	}
	var totalFailed int
	for _, pr := range results {
		filename := filepath.Base(pr.Hunt.SourcePath)
		fmt.Fprintf(os.Stderr, "\n%s\n📜 %s (worker %d)\n%s\n",
			strings.Repeat("=", 60), filename, pr.WorkerID, strings.Repeat("=", 60))
		if pr.Err != nil {
			logger.Error("hunt %q failed: %v", filename, pr.Err)
			totalFailed++
			continue
		}
		printResult(pr.Result, opts, logger)
		if hErr := report.AppendRunHistory("reports", pr.Result); hErr != nil {
			logger.Warn("run_history append failed: %v", hErr)
		}
		if cfg.HTMLReport && pr.Result != nil {
			reportPath, rErr := report.GenerateHTML(pr.Result, "reports")
			if rErr != nil {
				logger.Warn("HTML report generation failed: %v", rErr)
			} else {
				logger.Info("📊 HTML report: %s", reportPath)
			}
		}
		if pr.Result == nil || !pr.Result.Success {
			totalFailed++
		}
	}
	return totalFailed
}

// collectHuntFiles resolves a target path to a list of .hunt files.
func collectHuntFiles(target string) ([]string, error) {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("resolve path %q: %w", target, err)
	}

	info, err := os.Stat(absTarget)
	if err != nil {
		return nil, fmt.Errorf("path not found: %s", target)
	}

	if !info.IsDir() {
		if !strings.HasSuffix(absTarget, ".hunt") {
			return nil, fmt.Errorf("not a .hunt file: %s", target)
		}
		return []string{absTarget}, nil
	}

	// Collect all .hunt files in the directory (non-recursive).
	entries, err := os.ReadDir(absTarget)
	if err != nil {
		return nil, fmt.Errorf("read directory %q: %w", target, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".hunt") {
			files = append(files, filepath.Join(absTarget, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

// ── run-step subcommand ───────────────────────────────────────────────────────

func cmdRunStep(args []string) error {
	fs := flag.NewFlagSet("run-step", flag.ExitOnError)
	cdpEndpoint := fs.String("cdp", "http://127.0.0.1:9222", "CDP endpoint URL")
	verbose := fs.Bool("verbose", false, "enable verbose logging")
	jsonOut := fs.Bool("json", false, "print JSON result to stdout")
	compact := fs.Bool("compact", false, "emit the compact agent StepOutcome (reason + top candidates, no scorer breakdown) as JSON")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: driver run-step '<command>' [flags]\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	step := fs.Arg(0)
	if step == "" {
		fs.Usage()
		return fmt.Errorf("DSL command is required")
	}

	// --compact routes through pkg/agent so the CLI and embedded consumers
	// share one code path and one result shape.
	if *compact {
		return runStepCompact(*cdpEndpoint, step)
	}

	cfg := config.Default()
	cfg.CDPEndpoint = *cdpEndpoint
	cfg.Verbose = *verbose

	logLevel := utils.LogLevelInfo
	if *verbose {
		logLevel = utils.LogLevelDebug
	}
	// In -json mode, keep stdout clean for the payload.
	var logger *utils.Logger
	if *jsonOut {
		logger = utils.NewLoggerTo(os.Stderr, nil).WithLevel(logLevel)
	} else {
		logger = utils.NewLogger(nil).WithLevel(logLevel)
	}

	ctx := context.Background()

	b := browser.NewCDPBrowser(cfg.CDPEndpoint)
	page, err := b.FirstPage(ctx)
	if err != nil {
		return fmt.Errorf("connect to browser at %q: %w", cfg.CDPEndpoint, err)
	}
	defer page.Close()

	rt := runtime.New(cfg, page, logger)
	result, err := rt.RunStep(ctx, step)
	if err != nil {
		return err
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
		os.Stdout.Sync()
	} else {
		if result.Success {
			logger.Info("✓ %s", step)
		} else {
			logger.Error("✗ %s → %s", step, result.Error)
			return fmt.Errorf("step failed: %s", result.Error)
		}
	}

	return nil
}

// runStepCompact executes one step via pkg/agent and emits the compact
// StepOutcome JSON (ok/action/value/url/reason/score/near). On a failed step
// it still prints the outcome but returns an error so the process exits
// non-zero — letting a caller branch on exit code without parsing.
func runStepCompact(cdpEndpoint, step string) error {
	ctx := context.Background()
	sess, err := agent.Attach(ctx, cdpEndpoint, "", agent.Options{})
	if err != nil {
		return err
	}
	defer sess.Close()

	out, stepErr := sess.Step(ctx, step)
	if jErr := emitJSON(out); jErr != nil {
		return jErr
	}
	if stepErr != nil {
		// The outcome (with reason + near) is already on stdout; signal
		// failure via exit code without duplicating the message.
		return fmt.Errorf("step failed: %s", out.Reason)
	}
	return nil
}

// cmdRead implements `manul read "<target>" [--cdp ...] [--selector ...] [--json]`.
//
// It is the CLI face of agent.Session.Read / ReadText — a zero-scan way to pull
// one value (or a region's text) off the page an agent already has open. With
// --selector it returns the sanitized visible text of that CSS region (ReadText);
// without it, it resolves the human-labelled target and extracts its value (Read).
func cmdRead(args []string) error {
	fs := flag.NewFlagSet("read", flag.ExitOnError)
	cdpEndpoint := fs.String("cdp", "http://127.0.0.1:9222", "CDP endpoint URL")
	selector := fs.String("selector", "", "CSS selector for region text (uses ReadText instead of targeted Read)")
	urlSubstr := fs.String("tab", "", "attach to the page whose URL contains this substring")
	maxChars := fs.Int("max-chars", 0, "truncate --selector region text to this many characters (0 = no limit)")
	jsonOut := fs.Bool("json", false, "print JSON result to stdout")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: manul read '<target>' [flags]\n\n"+
			"  Reads one value off an already-open page (no full scan).\n"+
			"  With --selector, returns the sanitized text of that CSS region.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	target := fs.Arg(0)
	if target == "" && *selector == "" {
		fs.Usage()
		return fmt.Errorf("a target label or --selector is required")
	}

	ctx := context.Background()
	sess, err := agent.Attach(ctx, *cdpEndpoint, *urlSubstr, agent.Options{})
	if err != nil {
		return err
	}
	defer sess.Close()

	if *selector != "" {
		text, terr := sess.ReadText(ctx, *selector)
		if terr != nil {
			return terr
		}
		text = agent.TruncateText(text, *maxChars)
		if *jsonOut {
			return emitJSON(map[string]any{"text": text, "selector": *selector})
		}
		fmt.Println(text)
		return nil
	}

	v, rerr := sess.Read(ctx, target)
	if rerr != nil {
		return rerr
	}
	if *jsonOut {
		return emitJSON(map[string]any{"value": v.Text, "found": v.Found, "reason": string(v.Reason)})
	}
	if !v.Found {
		// Distinguish "nothing there" from an error: empty stdout, exit 0.
		return nil
	}
	fmt.Println(v.Text)
	return nil
}

// emitJSON writes v as indented JSON to stdout, keeping the payload clean.
func emitJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	os.Stdout.Sync()
	return nil
}

// ── Output helpers ────────────────────────────────────────────────────────────

func printResult(result any, opts outputOpts, logger *utils.Logger) {
	if opts.jsonl {
		// Streaming: per-step events were already emitted via the
		// runtime callback. Close out the stream with a single envelope
		// line carrying the aggregate HuntResult.
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(map[string]any{
			"event": "result",
			"data":  result,
		})
		os.Stdout.Sync()
		return
	}
	if opts.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
		os.Stdout.Sync()
		return
	}

	data, _ := json.Marshal(result)
	var s struct {
		TotalSteps      int   `json:"total_steps"`
		Passed          int   `json:"passed"`
		Failed          int   `json:"failed"`
		TotalDurationMS int64 `json:"total_duration_ms"`
		Success         bool  `json:"success"`
	}
	json.Unmarshal(data, &s)

	if s.Success {
		logger.Info("✓ All %d steps passed (%dms)", s.TotalSteps, s.TotalDurationMS)
		fmt.Fprintln(os.Stderr, "RESULT: PASS")
	} else {
		logger.Error("✗ %d/%d steps failed (%dms)", s.Failed, s.TotalSteps, s.TotalDurationMS)
		fmt.Fprintln(os.Stderr, "RESULT: FAIL")
	}
}

// cmdDaemon handles the `daemon` subcommand.
func cmdDaemon(args []string) error {
	fs := flag.NewFlagSet("daemon", flag.ExitOnError)
	headless := fs.Bool("headless", false, "run browser in headless mode")
	verbose := fs.Bool("verbose", false, "enable verbose logging")
	browserType := fs.String("browser", "chromium", "browser engine (chromium, firefox, webkit)")
	screenshot := fs.String("screenshot", "on-fail", "screenshot mode: none, on-fail, always")
	htmlReport := fs.Bool("html-report", false, "generate HTML report after each run")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := fs.Arg(0)
	if dir == "" {
		fs.Usage()
		return fmt.Errorf("directory path is required")
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cfg.Headless = *headless
	cfg.Verbose = *verbose
	cfg.Browser = *browserType
	cfg.Screenshot = *screenshot
	cfg.HTMLReport = *htmlReport
	logLevel := utils.LogLevelInfo
	if *verbose {
		logLevel = utils.LogLevelDebug
	}
	logger := utils.NewLogger(nil).WithLevel(logLevel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Handle Ctrl+C gracefully.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()
	return daemon.Run(ctx, dir, cfg, logger)
}

// cmdRecord handles the `record` subcommand.
func cmdRecord(args []string) error {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	output := fs.String("output", "tests/recorded_mission.hunt", "output file path")
	headless := fs.Bool("headless", false, "run browser in headless mode")
	if err := fs.Parse(args); err != nil {
		return err
	}
	url := fs.Arg(0)
	if url == "" {
		fs.Usage()
		return fmt.Errorf("URL is required")
	}
	return record.Run(context.Background(), url, *output, *headless)
}

// cmdScan handles the `scan` subcommand.
func cmdScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	output := fs.String("output", "draft.hunt", "output file for the draft (ignored in -json mode)")
	headless := fs.Bool("headless", false, "run browser in headless mode (ignored when --cdp is set)")
	full := fs.Bool("full", false, "full-page scan: group elements by semantic region (form, nav, main, shadow…)")
	cdpEndpoint := fs.String("cdp", "", "scan an already-loaded page via this CDP endpoint instead of launching Chrome (URL arg becomes optional)")
	jsonOut := fs.Bool("json", false, "emit the grouped scan result as JSON on stdout instead of writing a .hunt draft")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()

	// CDP path: probe an existing Chrome's current page. Used by external
	// drivers (e.g. OS-Manul) that own the browser lifecycle. URL arg is
	// optional because we don't navigate — we read whatever's loaded.
	if *cdpEndpoint != "" {
		if !*full {
			return fmt.Errorf("scan --cdp currently only supports --full mode")
		}
		groups, err := scan.ScanPageFullCDP(ctx, *cdpEndpoint)
		if err != nil {
			return err
		}
		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(compactScanGroups(groups))
		}
		// CDP + non-JSON: still write the .hunt draft so the human-driven
		// flow stays useful. URL isn't known at the CLI level, fall back
		// to a placeholder.
		huntText := scan.BuildHuntFull("about:current", groups)
		absOut, _ := filepath.Abs(*output)
		_ = os.MkdirAll(filepath.Dir(absOut), 0755)
		if err := os.WriteFile(absOut, []byte(huntText), 0644); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✅ Draft saved → %s\n", absOut)
		return nil
	}

	url := fs.Arg(0)
	if url == "" {
		fs.Usage()
		return fmt.Errorf("URL is required (or pass --cdp to scan an already-open page)")
	}

	if *jsonOut {
		// JSON over the launch-Chrome path: do the scan, emit JSON, skip
		// the .hunt write. Keeps stdout clean for downstream consumers.
		if !*full {
			return fmt.Errorf("scan -json currently only supports --full mode")
		}
		groups, err := scan.ScanPageFull(ctx, url, *headless)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(compactScanGroups(groups))
	}

	if *full {
		return scan.RunFull(ctx, url, *output, *headless)
	}
	return scan.Run(ctx, url, *output, *headless)
}

// cmdPages handles the `pages` subcommand.
func cmdPages(args []string) error {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage:\n  manul pages list\n  manul pages migrate <legacy-pages.json>\n")
		return fmt.Errorf("subcommand required")
	}
	switch args[0] {
	case "list":
		reg := pages.NewRegistry("")
		fmt.Printf("Page registry directory: %s\n", reg.Dir())
		// Reload and dump a summary.
		fmt.Println("(Use `cat pages/<site>.json` to inspect individual fragments)")
	case "migrate":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: manul pages migrate <legacy-pages.json>\n")
			return fmt.Errorf("legacy pages.json path required")
		}
		outDir := "pages"
		if err := pages.MigrateLegacyJSON(args[1], outDir); err != nil {
			return err
		}
		fmt.Printf("Migrated %s → %s/\n", args[1], outDir)
	default:
		fmt.Fprintf(os.Stderr, "unknown pages subcommand: %q\n\n", args[0])
		fmt.Fprintf(os.Stderr, "Usage:\n  manul pages list\n  manul pages migrate <legacy-pages.json>\n")
		return fmt.Errorf("unknown pages subcommand")
	}
	return nil
}

// cmdControls handles the `controls` subcommand.
func cmdControls(args []string) error {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage:\n  manul controls list\n")
		return fmt.Errorf("subcommand required")
	}
	switch args[0] {
	case "list":
		list := runtime.ListCustomControls()
		if len(list) == 0 {
			fmt.Println("No custom controls registered.")
			return nil
		}
		fmt.Printf("Registered custom controls (%d):\n", len(list))
		for _, entry := range list {
			fmt.Printf("  %-30s → %s\n", entry.Page, entry.Target)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown controls subcommand: %q\n\n", args[0])
		fmt.Fprintf(os.Stderr, "Usage:\n  manul controls list\n")
		return fmt.Errorf("unknown controls subcommand")
	}
	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Manul — deterministic DSL-first browser automation

Usage:
  manul <target> [flags]               Run .hunt files in target (file or directory)
  manul - [flags]                      Read a single hunt script from stdin
  manul run <target> [flags]           Explicit run subcommand
  manul run-step '<command>' [flags]   Execute a single DSL command (--compact for agent StepOutcome)
  manul read '<target>' [flags]        Read one value off an open page (zero-scan; --selector for region text)
  manul map [flags]                    Compact landmark-grouped JSON map of the open page (for LLM agents)
  manul schema [--json]                Emit the DSL grammar + agent JSON shapes as an LLM contract
  manul scan <URL> [flags]             Scan a URL and generate a draft .hunt file (--full for grouped scan)
  manul record <URL> [flags]           Record interactions and generate a .hunt file
  manul daemon <directory> [flags]     Run scheduled .hunt files continuously
  manul pages list                     List every site → pattern → label mapping under pages/
  manul pages migrate <file>           Split a legacy pages.json into pages/<site>.json fragments
  manul controls list                  List all registered @custom_control handlers

Core Flags:
  --cdp URL           Connect to existing Chrome (skip auto-launch)
  --user-data-dir DIR Chrome profile directory (default: unique temp dir per run)
  --headless          Run Chrome in headless mode
  --verbose           Enable verbose debug logging
  --json              Output structured JSON result to stdout
  --timeout DURATION  Per-command timeout (default: 30s)
  --tags TAGS         Filter hunt files by tags (comma-separated)
  --retries N         Retry failed hunt files up to N times (pass on retry = flaky)
  --screenshot MODE   Screenshot mode: on-fail (default), always, none
  --html-report       Generate HTML report after the run (default: true)
  --explain           Show targeting candidates (explain mode)
  --executable-path   Absolute path to a custom browser or Electron app executable

Daemon Flags:
  --headless          Run browser in headless mode
  --browser TYPE      Browser engine (default: chromium)
  --screenshot MODE   Screenshot mode for scheduled runs: on-fail, always, none
  --html-report       Generate HTML report after each scheduled run

Compatibility Flags:
  --workers N         Parallel workers (default: 1)
  --browser TYPE      Browser type (default: chromium)
  --break-lines L     Pause at specified line numbers (debugging)

Examples:
  manul examples/saucedemo.hunt
  manul examples/saucedemo.hunt --headless
  manul examples/
  manul .
  manul --workers 4 tests/
  manul --tags smoke tests/
  manul run-step "Click the 'Login' button" --cdp http://127.0.0.1:9222
  manul run-step "Click 'Checkout'" --compact --cdp http://127.0.0.1:9222
  manul read "Order total" --cdp http://127.0.0.1:9222
  manul read --selector "#answer" --max-chars 2000 --cdp http://127.0.0.1:9222
  manul map --cdp http://127.0.0.1:9222
  manul schema
`)
}
