package utils

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sync"
)

// LogLevel controls verbosity. Retained for backward-compatible call sites.
type LogLevel int

const (
	LogLevelInfo  LogLevel = 0
	LogLevelDebug LogLevel = 1
)

// StripANSIWriter wraps an io.Writer and strips ANSI color escape codes before
// each write so log files contain clean plain text.
type StripANSIWriter struct {
	W io.Writer
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Write strips ANSI codes from p and forwards the result to the underlying writer.
// Always returns len(p) to prevent spurious short-write errors.
func (s StripANSIWriter) Write(p []byte) (int, error) {
	_, err := s.W.Write(ansiRe.ReplaceAll(p, nil))
	return len(p), err
}

// Logger writes to os.Stdout (with ANSI colors) and optionally to a file writer
// (ANSI-stripped). All methods are safe for concurrent use.
type Logger struct {
	mu     *sync.Mutex // pointer so WithPrefix children share the parent's lock
	out    io.Writer
	file   io.Writer
	prefix string   // prepended inside messages, e.g. "[w3] "; set via WithPrefix
	level  LogLevel // retained for Debug gating
}

// NewLogger creates a Logger that always writes to os.Stdout.
// If logFile is non-nil, output is also written to it with ANSI codes stripped.
func NewLogger(logFile io.Writer) *Logger {
	return NewLoggerTo(os.Stdout, logFile)
}

// NewLoggerTo creates a Logger whose primary writer is out. Use this to
// route human-readable logs to os.Stderr when stdout must remain clean
// for a structured payload (e.g. the -json output that downstream
// consumers parse). If logFile is non-nil, output is mirrored to it with
// ANSI codes stripped.
func NewLoggerTo(out io.Writer, logFile io.Writer) *Logger {
	if out == nil {
		out = os.Stdout
	}
	l := &Logger{out: out, mu: &sync.Mutex{}}
	if logFile != nil {
		l.file = StripANSIWriter{logFile}
	}
	return l
}

// WithLevel returns a copy of l with the verbosity level set.
// Intended for enabling Debug output in --verbose mode.
// Example: logger := utils.NewLogger(nil).WithLevel(utils.LogLevelDebug)
func (l *Logger) WithLevel(level LogLevel) *Logger {
	return &Logger{mu: l.mu, out: l.out, file: l.file, prefix: l.prefix, level: level}
}

// WithPrefix returns a child Logger sharing the parent's writers and mutex,
// with prefix prepended to every line. Safe for concurrent use alongside siblings.
// If parent is nil, a default stdout/info logger is used as the base.
func WithPrefix(parent *Logger, prefix string) *Logger {
	if parent == nil {
		parent = NewLogger(nil)
	}
	return &Logger{
		mu:     parent.mu,
		out:    parent.out,
		file:   parent.file,
		prefix: parent.prefix + prefix,
		level:  parent.level,
	}
}

// write is the single choke-point for all output: acquires the lock, appends a
// newline, writes to the console writer, flushes stdout, then writes to the
// file writer if set. The Sync() call is required by the VS Code extension
// contract so every line is visible on the pipe immediately.
func (l *Logger) write(format string, args ...any) {
	msg := fmt.Sprintf(format, args...) + "\n"
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprint(l.out, msg)
	if f, ok := l.out.(*os.File); ok {
		f.Sync()
	}
	if l.file != nil {
		fmt.Fprint(l.file, msg)
	}
}

// ── Backward-compatible generic methods ──────────────────────────────────────

// Info logs a plain informational line (no ANSI).
func (l *Logger) Info(format string, args ...any) {
	l.write(l.prefix+format, args...)
}

// Debug logs a debug line; silenced when level < LogLevelDebug.
func (l *Logger) Debug(format string, args ...any) {
	if l.level >= LogLevelDebug {
		l.write(l.prefix+format, args...)
	}
}

// Warn logs a warning line in yellow.
func (l *Logger) Warn(format string, args ...any) {
	l.write("\033[33m"+l.prefix+format+"\033[0m", args...)
}

// Error logs an error line in red.
func (l *Logger) Error(format string, args ...any) {
	l.write("\033[31m"+l.prefix+format+"\033[0m", args...)
}

// ── Semantic visual methods ───────────────────────────────────────────────────
// Three-level visual hierarchy:
//
//	Block  (0 spaces): BlockStart / BlockPass / BlockFail
//	Action (2 spaces): ActionStart / ActionPass / ActionFail / ActionWarn
//	Detail (4 spaces): HeuristicDetail / ActionDetail

// Startup prints the 🐾 ManulEngine header line at mission start.
func (l *Logger) Startup(model, browser string) {
	l.write("\n🐾 ManulEngine [%s] | browser: %s", model, browser)
}

// BlockStart logs the 📦 BLOCK START banner (no indent).
func (l *Logger) BlockStart(name string) {
	l.write("\n[📦 BLOCK START] %s", name)
}

// BlockPass logs the 🟩 BLOCK PASS banner (no indent).
func (l *Logger) BlockPass(name string) {
	l.write("[🟩 BLOCK PASS] %s", name)
}

// BlockFail logs the 🟥 BLOCK FAIL banner (no indent).
func (l *Logger) BlockFail(name string) {
	l.write("[🟥 BLOCK FAIL] %s", name)
}

// ActionStart logs the ▶️ ACTION START line (2-space indent).
func (l *Logger) ActionStart(step string) {
	l.write("  [▶️ ACTION START] %s", step)
}

// ActionPass logs the ✅ ACTION PASS line (2-space indent).
// durationSec is the wall-clock elapsed time in seconds.
func (l *Logger) ActionPass(durationSec float64) {
	l.write("  [✅ ACTION PASS] duration: %.2fs", durationSec)
}

// ActionFail logs the ❌ ACTION FAIL line (2-space indent).
func (l *Logger) ActionFail(err error) {
	l.write("  [❌ ACTION FAIL] %v", err)
}

// ActionWarn logs the ⚠️ ACTION WARN line (2-space indent).
func (l *Logger) ActionWarn(msg string) {
	l.write("  [⚠️ ACTION WARN] %s", msg)
}

// HeuristicDetail logs a ⚙️ DOM HEURISTICS detail line (4-space indent).
func (l *Logger) HeuristicDetail(confidence float64, details string) {
	l.write("    ⚙️  DOM HEURISTICS: %s (confidence %.3f)", details, confidence)
}

// ActionDetail logs a freeform detail line (4-space indent) with a custom emoji.
// Example: logger.ActionDetail("⌨️", "Typed %q → %q", value, elementName)
func (l *Logger) ActionDetail(emoji string, format string, args ...any) {
	body := fmt.Sprintf(format, args...)
	l.write("    %s  %s", emoji, body)
}
