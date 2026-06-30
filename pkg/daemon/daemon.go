// Package daemon implements the `manul daemon <directory>` subcommand.
//
// It monitors a directory for .hunt files with @schedule: directives and
// runs them on their specified cadence.
package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexbeatnik/ManulEngineGo/pkg/browser"
	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/report"
	"github.com/alexbeatnik/ManulEngineGo/pkg/runtime"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
)

var weekdays = map[string]int{
	"monday": 0, "tuesday": 1, "wednesday": 2, "thursday": 3,
	"friday": 4, "saturday": 5, "sunday": 6,
}

// Schedule is a parsed @schedule expression.
type Schedule struct {
	Raw               string
	IntervalSeconds   int
	DailyAt           *time.Time
	WeeklyDay         int
	WeeklyHour        int
	WeeklyMinute      int
}

// ParseSchedule converts expressions like "every 5 minutes" into a Schedule.
func ParseSchedule(expr string) (*Schedule, error) {
	s := strings.ToLower(strings.TrimSpace(expr))
	if s == "" {
		return nil, fmt.Errorf("empty schedule expression")
	}

	// every N minutes/seconds/hours
	reEveryN := regexp.MustCompile(`^every\s+(\d+)\s+(second|seconds|minute|minutes|hour|hours)$`)
	if m := reEveryN.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		unit := strings.TrimSuffix(m[2], "s")
		multiplier := map[string]int{"second": 1, "minute": 60, "hour": 3600}[unit]
		return &Schedule{Raw: expr, IntervalSeconds: n * multiplier}, nil
	}

	// every minute/hour/second
	reEveryUnit := regexp.MustCompile(`^every\s+(second|minute|hour)$`)
	if m := reEveryUnit.FindStringSubmatch(s); m != nil {
		multiplier := map[string]int{"second": 1, "minute": 60, "hour": 3600}[m[1]]
		return &Schedule{Raw: expr, IntervalSeconds: multiplier}, nil
	}

	// daily at HH:MM
	reDaily := regexp.MustCompile(`^daily\s+at\s+(\d{1,2}):(\d{2})$`)
	if m := reDaily.FindStringSubmatch(s); m != nil {
		hh, _ := strconv.Atoi(m[1])
		mm, _ := strconv.Atoi(m[2])
		if hh < 0 || hh > 23 || mm < 0 || mm > 59 {
			return nil, fmt.Errorf("invalid time in schedule: %q", expr)
		}
		t := time.Date(0, 1, 1, hh, mm, 0, 0, time.Local)
		return &Schedule{Raw: expr, DailyAt: &t}, nil
	}

	// every monday [at HH:MM]
	reWeekly := regexp.MustCompile(`^every\s+(` + strings.Join(keys(weekdays), "|") + `)(?:\s+at\s+(\d{1,2}):(\d{2}))?$`)
	if m := reWeekly.FindStringSubmatch(s); m != nil {
		day := weekdays[m[1]]
		hh, mm := 0, 0
		if m[2] != "" {
			hh, _ = strconv.Atoi(m[2])
			mm, _ = strconv.Atoi(m[3])
		}
		return &Schedule{Raw: expr, WeeklyDay: day, WeeklyHour: hh, WeeklyMinute: mm}, nil
	}

	return nil, fmt.Errorf("unrecognised schedule expression: %q", expr)
}

func keys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// NextRunDelay returns seconds until the next scheduled execution.
func NextRunDelay(s *Schedule) time.Duration {
	now := time.Now()
	if s.IntervalSeconds > 0 {
		return time.Duration(s.IntervalSeconds) * time.Second
	}
	if s.DailyAt != nil {
		target := time.Date(now.Year(), now.Month(), now.Day(), s.DailyAt.Hour(), s.DailyAt.Minute(), 0, 0, now.Location())
		if !target.After(now) {
			target = target.Add(24 * time.Hour)
		}
		return target.Sub(now)
	}
	if s.WeeklyDay >= 0 {
		daysAhead := (s.WeeklyDay - int(now.Weekday()) + 7) % 7
		target := time.Date(now.Year(), now.Month(), now.Day(), s.WeeklyHour, s.WeeklyMinute, 0, 0, now.Location())
		target = target.Add(time.Duration(daysAhead) * 24 * time.Hour)
		if !target.After(now) {
			target = target.Add(7 * 24 * time.Hour)
		}
		return target.Sub(now)
	}
	return time.Hour
}

// ScheduledHunt is a hunt file that has a @schedule directive.
type ScheduledHunt struct {
	Path     string
	Schedule *Schedule
	Hunt     *dsl.Hunt
}

// CollectScheduledHunts recursively scans dir for .hunt files with @schedule: headers.
func CollectScheduledHunts(dir string) ([]ScheduledHunt, error) {
	var out []ScheduledHunt
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".hunt") {
			return nil
		}
		hunt, err := dsl.ParseFile(path)
		if err != nil {
			return nil
		}
		if hunt.Schedule != "" {
			sched, err := ParseSchedule(hunt.Schedule)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  %s: invalid @schedule: %v\n", d.Name(), err)
				return nil
			}
			out = append(out, ScheduledHunt{Path: path, Schedule: sched, Hunt: hunt})
		}
		return nil
	})
	return out, err
}

// Run starts the daemon loop for scheduled hunts in dir.
func Run(ctx context.Context, dir string, cfg config.Config, logger *utils.Logger) error {
	hunts, err := CollectScheduledHunts(dir)
	if err != nil {
		return err
	}
	if len(hunts) == 0 {
		return fmt.Errorf("no scheduled .hunt files found in %q", dir)
	}

	fmt.Fprintf(os.Stderr, "👹 Manul Daemon — %d scheduled hunt(s) in %s\n", len(hunts), dir)

	var wg sync.WaitGroup
	for _, h := range hunts {
		wg.Add(1)
		go func(sh ScheduledHunt) {
			defer wg.Done()
			runJob(ctx, sh, cfg, logger)
		}(h)
	}

	<-ctx.Done()
	fmt.Fprintln(os.Stderr, "\n🛑 Daemon shutting down…")
	wg.Wait()
	return nil
}

func runJob(ctx context.Context, sh ScheduledHunt, cfg config.Config, logger *utils.Logger) {
	filename := filepath.Base(sh.Path)
	for {
		delay := NextRunDelay(sh.Schedule)
		next := time.Now().Add(delay)
		fmt.Fprintf(os.Stderr, "⏰ [%s] next run at %s (in %.0fs) — %s\n",
			filename, next.Format("2006-01-02 15:04:05"), delay.Seconds(), sh.Schedule.Raw)

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		fmt.Fprintf(os.Stderr, "\n🚀 [%s] scheduled run starting — %s\n",
			filename, time.Now().Format("15:04:05"))

		if err := executeHunt(ctx, sh.Hunt, cfg, logger); err != nil {
			fmt.Fprintf(os.Stderr, "💥 [%s] crashed — %v\n", filename, err)
		} else {
			fmt.Fprintf(os.Stderr, "🏁 [%s] finished\n", filename)
		}
	}
}

func executeHunt(ctx context.Context, hunt *dsl.Hunt, cfg config.Config, logger *utils.Logger) error {
	opts := browser.DefaultChromeOptions()
	opts.Headless = cfg.Headless
	chrome, err := browser.LaunchChrome(ctx, opts)
	if err != nil {
		return fmt.Errorf("launch chrome: %w", err)
	}
	defer chrome.Close()

	b := browser.NewCDPBrowser(chrome.Endpoint())
	page, err := b.FirstPage(ctx)
	if err != nil {
		return fmt.Errorf("connect to page: %w", err)
	}
	defer page.Close()

	rt := runtime.New(cfg, page, logger)
	result, err := rt.RunHunt(ctx, hunt)
	if err != nil {
		return err
	}
	_ = report.AppendRunHistory("reports", result)
	if cfg.HTMLReport {
		_, _ = report.GenerateHTML(result, "reports")
	}
	return nil
}
