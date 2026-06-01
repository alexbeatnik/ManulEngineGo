// Package record implements the `manul record <URL>` subcommand for ManulHeart.
//
// It opens a URL in Chrome, injects a JS recorder, waits for the user to
// finish interacting, then writes a .hunt file from the captured events.
package record

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexbeatnik/ManulHeart/pkg/browser"
)

// RECORDER_JS is injected into the page to capture user interactions.
const RECORDER_JS = `(function(){
  if (window.__manulRecorder) return;
  window.__manulRecorder = true;
  window.__manulEvents = [];
  function push(type, data) {
    window.__manulEvents.push({type: type, data: data, time: Date.now()});
  }
  document.addEventListener('click', function(e){
    var el = e.target;
    var text = (el.innerText || el.textContent || '').replace(/\s+/g, ' ').trim();
    if (!text) text = el.getAttribute('aria-label') || el.getAttribute('title') || '';
    if (text && text.length <= 80) push('click', {target: text, tag: el.tagName});
  }, true);
  document.addEventListener('input', function(e){
    var el = e.target;
    var label = el.getAttribute('aria-label') || el.getAttribute('placeholder') || el.getAttribute('name') || '';
    if (label) push('fill', {target: label, value: el.value});
  }, true);
  document.addEventListener('keydown', function(e){
    if (e.key === 'Enter') push('press', {key: 'Enter'});
  }, true);
  console.log('🎬 Manul recorder active — interact with the page, then press Enter in the terminal to stop.');
})();`

// Event is a single recorded interaction.
type Event struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
	Time int64                  `json:"time"`
}

func eventToDSL(ev Event) string {
	switch ev.Type {
	case "click":
		target := fmt.Sprint(ev.Data["target"])
		tag := fmt.Sprint(ev.Data["tag"])
		if strings.ToUpper(tag) == "A" {
			return fmt.Sprintf("Click the '%s' link", target)
		}
		return fmt.Sprintf("Click the '%s' button", target)
	case "fill":
		target := fmt.Sprint(ev.Data["target"])
		value := fmt.Sprint(ev.Data["value"])
		return fmt.Sprintf("Fill '%s' field with '%s'", target, value)
	case "press":
		key := fmt.Sprint(ev.Data["key"])
		return fmt.Sprintf("PRESS %s", key)
	default:
		return ""
	}
}

func buildHunt(url string, events []Event) string {
	lines := []string{
		fmt.Sprintf("@context: Auto-generated recording for %s", url),
		"@title: recorded-mission",
		"",
		fmt.Sprintf("STEP 1:\n    NAVIGATE to %s", url),
		"",
	}
	step := 2
	seen := make(map[string]bool)
	for _, ev := range events {
		action := eventToDSL(ev)
		if action == "" {
			continue
		}
		key := ev.Type + "|" + fmt.Sprint(ev.Data["target"])
		if seen[key] && ev.Type == "fill" {
			// Update previous fill instead of duplicating
			continue
		}
		seen[key] = true
		lines = append(lines, fmt.Sprintf("STEP %d:\n    %s", step, action))
		lines = append(lines, "")
		step++
	}
	lines = append(lines, "DONE.")
	return strings.Join(lines, "\n") + "\n"
}

// Run is the entry point for `manul record <URL>`.
func Run(ctx context.Context, url, outputFile string, headless bool) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	opts := browser.DefaultChromeOptions()
	opts.Headless = headless
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

	fmt.Printf("\n🎬 Manul Recorder — recording %s\n", url)
	fmt.Println("   Browser: chromium | Headless:", headless)
	fmt.Println("   Interact with the page, then press Enter in the terminal to stop.")

	if err := page.Navigate(ctx, url); err != nil {
		return fmt.Errorf("navigate: %w", err)
	}
	_ = page.Wait(ctx, 2*time.Second)
	_ = page.WaitForLoad(ctx)

	// Inject recorder JS
	_, err = page.EvalJS(ctx, RECORDER_JS)
	if err != nil {
		return fmt.Errorf("inject recorder: %w", err)
	}

	// Wait for user to press Enter
	fmt.Println("\n   Recording… (press Enter to stop)")
	bufio.NewReader(os.Stdin).ReadString('\n')

	// Read recorded events
	raw, err := page.EvalJS(ctx, `JSON.stringify(window.__manulEvents || [])`)
	if err != nil {
		return fmt.Errorf("read events: %w", err)
	}

	var events []Event
	if err := json.Unmarshal(raw, &events); err != nil {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			_ = json.Unmarshal([]byte(s), &events)
		}
	}
	fmt.Printf("   Captured %d event(s).\n", len(events))

	huntText := buildHunt(url, events)
	absOut, _ := filepath.Abs(outputFile)
	_ = os.MkdirAll(filepath.Dir(absOut), 0755)
	if err := os.WriteFile(absOut, []byte(huntText), 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("\n✅ Recording saved → %s\n", absOut)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(huntText)
	fmt.Println(strings.Repeat("─", 60))
	return nil
}
