// Package scan implements the `manul scan <URL>` subcommand for ManulEngine (Go).
//
// It opens a URL in Chrome, runs a DOM scanner JS, and writes a draft .hunt file.
package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alexbeatnik/ManulEngineGo/pkg/browser"
)

// SCAN_JS is the JavaScript payload executed in the browser to discover
// interactive elements. Mirrors ManulEngine's SCAN_JS.
const SCAN_JS = `() => {
    function isHidden(el) {
        if (el.getAttribute('aria-hidden') === 'true') return true;
        const r = el.getBoundingClientRect();
        if (r.width === 0 && r.height === 0) return true;
        try {
            const st = window.getComputedStyle(el);
            if (st.display === 'none' || st.visibility === 'hidden' || parseFloat(st.opacity) === 0) return true;
        } catch (_) {}
        return false;
    }
    function bestLabel(el) {
        const tag  = el.tagName ? el.tagName.toUpperCase() : '';
        const type = (el.getAttribute('type') || '').toLowerCase();
        if (tag === 'INPUT' && (type === 'radio' || type === 'checkbox')) {
            if (el.id) {
                const root = el.getRootNode();
                const lbl = root.querySelector('label[for="' + CSS.escape(el.id) + '"]');
                if (lbl) return lbl.innerText.trim();
            }
            const closestLbl = el.closest('label');
            if (closestLbl) return closestLbl.innerText.trim();
            const nextSib = el.nextElementSibling;
            if (nextSib && nextSib.tagName === 'LABEL') return nextSib.innerText.trim();
        }
        const text = (el.innerText || el.textContent || '').replace(/\s+/g, ' ').trim();
        if (text && text.length <= 80) return text;
        const aria = el.getAttribute('aria-label') || '';
        if (aria.trim()) return aria.trim();
        const ph = el.getAttribute('placeholder') || '';
        if (ph.trim()) return ph.trim();
        const title = el.getAttribute('title') || '';
        if (title.trim()) return title.trim();
        const name = el.getAttribute('name') || '';
        if (name.trim()) return name.trim();
        const id = el.getAttribute('id') || '';
        if (id.trim()) return id.trim();
        return '';
    }
    function classify(el) {
        const tag = el.tagName ? el.tagName.toUpperCase() : '';
        const type = (el.getAttribute('type') || '').toLowerCase();
        const role = (el.getAttribute('role') || '').toLowerCase();
        if (tag === 'SELECT') return 'select';
        if (tag === 'INPUT' && type === 'checkbox') return 'checkbox';
        if (tag === 'INPUT' && type === 'radio') return 'radio';
        if (tag === 'INPUT' && !['submit', 'reset', 'image', 'hidden', 'button'].includes(type)) return 'input';
        if (tag === 'TEXTAREA') return 'input';
        if (tag === 'BUTTON') return 'button';
        if (tag === 'A' && el.getAttribute('href') !== null) return 'link';
        if (role === 'button') return 'button';
        if (role === 'link') return 'link';
        if (role === 'checkbox') return 'checkbox';
        if (role === 'radio') return 'radio';
        if (role === 'combobox') return 'select';
        if (role === 'switch') return 'checkbox';
        if (tag === 'INPUT' && type === 'submit') return 'button';
        if (tag === 'INPUT' && type === 'button') return 'button';
        return null;
    }
    function scanRoot(root, results, seen) {
        const candidates = root.querySelectorAll(
            'button, a[href], input, select, textarea, ' +
            '[role="button"], [role="link"], [role="checkbox"], [role="radio"], ' +
            '[role="combobox"], [role="switch"]'
        );
        for (const el of candidates) {
            if (seen.has(el)) continue;
            seen.add(el);
            if (isHidden(el)) continue;
            const kind = classify(el);
            if (!kind) continue;
            const label = bestLabel(el);
            if (!label) continue;
            const entry = { type: kind, identifier: label };
            if ((kind === 'input' || kind === 'select') && el.value !== undefined && el.value !== '') {
                entry.value = el.value;
            }
            results.push(entry);
            if (el.shadowRoot) scanRoot(el.shadowRoot, results, seen);
        }
    }
    const results = [];
    const seen = new Set();
    scanRoot(document, results, seen);
    return JSON.stringify(results);
}`

var skipLabels = map[string]bool{
	"": true, "click": true, "button": true, "submit": true, "link": true,
	"go": true, "close": true, "×": true, "✕": true, "✖": true,
	"menu": true, "toggle": true, "show": true, "hide": true,
}

// Element represents a scanned interactive element.
type Element struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
	Value      string `json:"value,omitempty"`
}

func isUseful(identifier, kind string) bool {
	label := strings.TrimSpace(strings.ToLower(identifier))
	if label == "" || skipLabels[label] {
		return false
	}
	if len(label) > 80 {
		return false
	}
	if strings.HasPrefix(label, "http://") || strings.HasPrefix(label, "https://") {
		return false
	}
	return true
}

func mapToStep(kind, identifier string) string {
	i := strings.TrimSpace(identifier)
	switch kind {
	case "input", "textbox":
		return fmt.Sprintf("Fill '%s' with ''", i)
	case "select", "combobox", "listbox":
		return fmt.Sprintf("Select 'Option' from the '%s' dropdown", i)
	case "checkbox", "switch":
		return fmt.Sprintf("Check the checkbox for '%s'", i)
	case "radio":
		return fmt.Sprintf("Click the radio button for '%s'", i)
	case "link":
		return fmt.Sprintf("Click the '%s' link", i)
	default:
		return fmt.Sprintf("Click the '%s' button", i)
	}
}

// BuildHunt generates a draft .hunt file from a URL and scanned elements.
func BuildHunt(url string, elements []Element) string {
	lines := []string{
		fmt.Sprintf("@context: Auto-generated scan for %s", url),
		"@title: scan-draft",
		"",
		fmt.Sprintf("STEP 1:\n    NAVIGATE to %s", url),
		"",
		"STEP 2:\n    WAIT 2",
		"",
	}

	step := 3
	seen := make(map[string]bool)

	for _, el := range elements {
		if !isUseful(el.Identifier, el.Type) {
			continue
		}
		key := el.Type + "|" + strings.ToLower(el.Identifier)
		if seen[key] {
			continue
		}
		seen[key] = true

		action := mapToStep(el.Type, el.Identifier)
		lines = append(lines, fmt.Sprintf("STEP %d:\n    %s", step, action))
		lines = append(lines, "")
		step++
	}

	lines = append(lines, "DONE.")
	return strings.Join(lines, "\n") + "\n"
}

// ScanPage opens url in a headless Chrome, runs the DOM scanner, and returns
// the scanned elements.
func ScanPage(ctx context.Context, url string, headless bool) ([]Element, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	opts := browser.DefaultChromeOptions()
	opts.Headless = headless
	chrome, err := browser.LaunchChrome(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("launch chrome: %w", err)
	}
	defer chrome.Close()

	b := browser.NewCDPBrowser(chrome.Endpoint())
	page, err := b.FirstPage(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to page: %w", err)
	}
	defer page.Close()

	if err := page.Navigate(ctx, url); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}
	_ = page.Wait(ctx, 2*time.Second)
	_ = page.WaitForLoad(ctx)

	raw, err := page.EvalJS(ctx, SCAN_JS)
	if err != nil {
		return nil, fmt.Errorf("scan js: %w", err)
	}

	var elements []Element
	if err := json.Unmarshal(raw, &elements); err != nil {
		// SCAN_JS returns JSON string, try unmarshal again
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			_ = json.Unmarshal([]byte(s), &elements)
		}
	}
	return elements, nil
}

// FULL_SCAN_JS discovers all interactive elements grouped by semantic page regions
// (forms, navigation, main, dialog, etc.) including Shadow DOM. Mirrors ManulEngine's FULL_SCAN_JS.
const FULL_SCAN_JS = `() => {
    function isHidden(el) {
        if (el.getAttribute('aria-hidden') === 'true') return true;
        const r = el.getBoundingClientRect();
        if (r.width === 0 && r.height === 0) return true;
        try {
            const st = window.getComputedStyle(el);
            if (st.display === 'none' || st.visibility === 'hidden' || parseFloat(st.opacity) === 0) return true;
        } catch (_) {}
        return false;
    }
    function bestLabel(el) {
        const tag  = (el.tagName || '').toUpperCase();
        const type = (el.getAttribute('type') || '').toLowerCase();
        if (tag === 'INPUT' && (type === 'radio' || type === 'checkbox')) {
            if (el.id) {
                const lbl = document.querySelector('label[for="' + CSS.escape(el.id) + '"]');
                if (lbl) return lbl.innerText.trim();
            }
            const closest = el.closest('label');
            if (closest) return closest.innerText.trim();
        }
        const ariaLabel = el.getAttribute('aria-label') || '';
        if (ariaLabel.trim()) return ariaLabel.trim();
        const ph = el.getAttribute('placeholder') || '';
        if (ph.trim()) return ph.trim();
        const text = (el.innerText || el.textContent || '').replace(/\s+/g, ' ').trim();
        if (text && text.length <= 80) return text;
        const title = el.getAttribute('title') || '';
        if (title.trim()) return title.trim();
        const name = el.getAttribute('name') || '';
        if (name.trim()) return name.trim();
        return el.getAttribute('id') || '';
    }
    function bestLocator(el) {
        const id = el.getAttribute('id');
        if (id) return '#' + CSS.escape(id);
        const testId = el.getAttribute('data-testid') || el.getAttribute('data-test-id');
        if (testId) return '[data-testid="' + testId + '"]';
        const label = bestLabel(el).slice(0, 60);
        if (label) return 'text=' + label;
        const name = el.getAttribute('name');
        if (name) return '[name="' + name + '"]';
        return (el.tagName || '').toLowerCase();
    }
    function elementRole(el) {
        const role = (el.getAttribute('role') || '').toLowerCase();
        if (role) return role;
        const tag  = (el.tagName || '').toUpperCase();
        const type = (el.getAttribute('type') || '').toLowerCase();
        if (tag === 'BUTTON' || type === 'submit' || type === 'button') return 'button';
        if (tag === 'A') return 'link';
        if (tag === 'INPUT' && type === 'checkbox') return 'checkbox';
        if (tag === 'INPUT' && type === 'radio') return 'radio';
        if (tag === 'SELECT') return 'combobox';
        if (tag === 'TEXTAREA' || tag === 'INPUT') return 'textbox';
        return (tag || '').toLowerCase();
    }
    function isInteractive(el) {
        const tag  = (el.tagName || '').toUpperCase();
        const type = (el.getAttribute('type') || '').toLowerCase();
        if (['BUTTON', 'SELECT', 'TEXTAREA'].includes(tag)) return true;
        if (tag === 'INPUT' && type !== 'hidden') return true;
        if (tag === 'A' && el.getAttribute('href') !== null) return true;
        const role = (el.getAttribute('role') || '').toLowerCase();
        return ['button','link','checkbox','radio','combobox','switch','menuitem',
                'tab','option','menuitemcheckbox','menuitemradio'].includes(role);
    }

    const LANDMARK_ROLES = new Set(['form','navigation','search','banner','contentinfo',
                                    'complementary','main','region','dialog']);
    const LANDMARK_TAGS  = new Set(['FORM','NAV','HEADER','FOOTER','ASIDE','MAIN','DIALOG',
                                    'SECTION','ARTICLE']);

    function groupLabel(container) {
        const al = container.getAttribute('aria-label') || '';
        if (al.trim()) return al.trim();
        const lby = container.getAttribute('aria-labelledby') || '';
        if (lby) {
            const ref = document.getElementById(lby);
            if (ref) return (ref.innerText || ref.textContent || '').trim();
        }
        if (container.tagName === 'FIELDSET') {
            const legend = container.querySelector('legend');
            if (legend) return legend.innerText.trim();
        }
        const h = container.querySelector('h1,h2,h3,h4,h5,h6');
        if (h && h.innerText.trim()) return h.innerText.trim().slice(0, 60);
        const id = container.getAttribute('id') || '';
        if (id) {
            const lbl = document.querySelector('label[for="' + CSS.escape(id) + '"]');
            if (lbl) return lbl.innerText.trim();
        }
        return '';
    }

    function resolveGroup(el) {
        let node = el.parentElement;
        while (node && node !== document.body) {
            const tag  = (node.tagName || '').toUpperCase();
            const role = (node.getAttribute('role') || '').toLowerCase();
            if (LANDMARK_TAGS.has(tag) || LANDMARK_ROLES.has(role)) {
                const label = groupLabel(node);
                const tagName = tag.charAt(0) + tag.slice(1).toLowerCase();
                if (label) return tagName + ': ' + label;
                return tagName;
            }
            node = node.parentElement;
        }
        return 'Page';
    }

    const seen   = new WeakSet();
    const groups = {};

    const SELECTOR =
        'button, a[href], input:not([type=hidden]), select, textarea, ' +
        '[role="button"], [role="link"], [role="checkbox"], [role="radio"], ' +
        '[role="combobox"], [role="switch"], [role="menuitem"], [role="tab"]';

    function processElement(el, inShadow) {
        if (seen.has(el) || isHidden(el)) return;
        if (!isInteractive(el)) return;
        seen.add(el);
        const label = bestLabel(el);
        if (!label) return;
        const role     = elementRole(el);
        const locator  = bestLocator(el);
        const tag      = (el.tagName || '').toLowerCase();
        const editable = ['textbox','combobox'].includes(role) ||
                         tag === 'textarea' ||
                         (tag === 'input' && !['checkbox','radio','button','submit','image'].includes(
                             (el.getAttribute('type') || '').toLowerCase()));
        const group    = resolveGroup(el);
        const suffix   = inShadow ? ' [shadow]' : '';
        if (!groups[group + suffix]) groups[group + suffix] = [];
        groups[group + suffix].push({ role, label, locator, tag, editable });
    }

    function scanRoot(root, inShadow) {
        for (const el of root.querySelectorAll(SELECTOR)) {
            processElement(el, inShadow);
        }
        for (const el of root.querySelectorAll('*')) {
            if (el.shadowRoot) scanRoot(el.shadowRoot, true);
        }
    }

    scanRoot(document, false);
    return JSON.stringify(groups);
}`

// FullElement represents one interactive element from a full-page scan.
type FullElement struct {
	Role     string `json:"role"`
	Label    string `json:"label"`
	Locator  string `json:"locator"`
	Tag      string `json:"tag"`
	Editable bool   `json:"editable"`
}

// BuildHuntFull generates a draft .hunt file from grouped full-page scan results.
// Groups from Shadow DOM roots are annotated with [shadow].
func BuildHuntFull(url string, groups map[string][]FullElement) string {
	lines := []string{
		fmt.Sprintf("@context: Auto-generated full scan for %s", url),
		"@title: scan-full-draft",
		"",
		fmt.Sprintf("STEP 1:\n    NAVIGATE to %s", url),
		"",
		"STEP 2:\n    WAIT 2",
		"",
	}

	step := 3
	seen := make(map[string]bool)

	// Stable group order: Page first, then alphabetically.
	groupNames := make([]string, 0, len(groups))
	for g := range groups {
		groupNames = append(groupNames, g)
	}
	sort.Slice(groupNames, func(i, j int) bool {
		if groupNames[i] == "Page" {
			return true
		}
		if groupNames[j] == "Page" {
			return false
		}
		return groupNames[i] < groupNames[j]
	})

	for _, groupName := range groupNames {
		els := groups[groupName]
		if len(els) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("# ── %s ──", groupName), "")
		for _, el := range els {
			if !isUseful(el.Label, el.Role) {
				continue
			}
			key := el.Role + "|" + strings.ToLower(el.Label)
			if seen[key] {
				continue
			}
			seen[key] = true

			action := mapToStep(el.Role, el.Label)
			lines = append(lines, fmt.Sprintf("STEP %d:\n    %s", step, action))
			lines = append(lines, "")
			step++
		}
	}

	lines = append(lines, "DONE.")
	return strings.Join(lines, "\n") + "\n"
}

// ScanPageFull opens url in Chrome, runs the full-page grouped DOM scanner
// (including Shadow DOM), and returns elements grouped by semantic region.
func ScanPageFull(ctx context.Context, url string, headless bool) (map[string][]FullElement, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	opts := browser.DefaultChromeOptions()
	opts.Headless = headless
	chrome, err := browser.LaunchChrome(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("launch chrome: %w", err)
	}
	defer chrome.Close()

	b := browser.NewCDPBrowser(chrome.Endpoint())
	page, err := b.FirstPage(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to page: %w", err)
	}
	defer page.Close()

	if err := page.Navigate(ctx, url); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}
	_ = page.Wait(ctx, 2*time.Second)
	_ = page.WaitForLoad(ctx)

	return ScanCurrentPageFull(ctx, page)
}

// ScanCurrentPageFull runs the full-page scan against an already-connected
// Page. Use when you've already navigated (typically because the page is
// being driven through an existing CDP session) and just want the
// landmark-grouped element map for the page that's currently visible.
//
// Same return shape as ScanPageFull, but with no Chrome lifecycle and no
// navigation — strictly a JS probe on whatever's loaded right now.
func ScanCurrentPageFull(ctx context.Context, page browser.Page) (map[string][]FullElement, error) {
	raw, err := page.EvalJS(ctx, FULL_SCAN_JS)
	if err != nil {
		return nil, fmt.Errorf("full scan js: %w", err)
	}

	groups := make(map[string][]FullElement)
	if err := json.Unmarshal(raw, &groups); err != nil {
		// FULL_SCAN_JS may return a JSON string in some CDP wrappers — unwrap it.
		var s string
		if err2 := json.Unmarshal(raw, &s); err2 == nil {
			_ = json.Unmarshal([]byte(s), &groups)
		}
	}
	return groups, nil
}

// ScanPageFullCDP runs the full-page scan against an external Chrome via
// its CDP endpoint, without launching or navigating. Attaches to the first
// available page target and probes whatever's currently loaded.
//
// This is the integration entry point for callers (OS-Manul, IDE
// extensions, …) that own the Chrome lifecycle themselves and want the
// landmark map of the page their user is looking at right now.
func ScanPageFullCDP(ctx context.Context, cdpEndpoint string) (map[string][]FullElement, error) {
	if cdpEndpoint == "" {
		return nil, fmt.Errorf("scan: cdp endpoint required")
	}
	b := browser.NewCDPBrowser(cdpEndpoint)
	page, err := b.FirstPage(ctx)
	if err != nil {
		return nil, fmt.Errorf("scan: connect to page at %q: %w", cdpEndpoint, err)
	}
	defer page.Close()
	return ScanCurrentPageFull(ctx, page)
}

// RunFull is the entry point for `manul scan --full <URL>`.
func RunFull(ctx context.Context, url, outputFile string, headless bool) error {
	fmt.Printf("\n🔍 Manul Full Scanner — scanning %s\n", url)
	fmt.Printf("   Headless: %v\n", headless)

	groups, err := ScanPageFull(ctx, url, headless)
	if err != nil {
		return err
	}
	total := 0
	for _, els := range groups {
		total += len(els)
	}
	fmt.Printf("   Found %d interactive element(s) in %d group(s) before dedup/filter.\n", total, len(groups))

	huntText := BuildHuntFull(url, groups)

	absOut, _ := filepath.Abs(outputFile)
	_ = os.MkdirAll(filepath.Dir(absOut), 0755)
	if err := os.WriteFile(absOut, []byte(huntText), 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("\n✅ Draft saved → %s\n", absOut)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(huntText)
	fmt.Println(strings.Repeat("─", 60))
	return nil
}

// Run is the entry point for `manul scan <URL>`.
func Run(ctx context.Context, url, outputFile string, headless bool) error {
	fmt.Printf("\n🔍 Manul Scanner — scanning %s\n", url)
	fmt.Printf("   Headless: %v\n", headless)

	elements, err := ScanPage(ctx, url, headless)
	if err != nil {
		return err
	}
	fmt.Printf("   Found %d interactive element(s) before dedup/filter.\n", len(elements))

	huntText := BuildHunt(url, elements)

	absOut, _ := filepath.Abs(outputFile)
	_ = os.MkdirAll(filepath.Dir(absOut), 0755)
	if err := os.WriteFile(absOut, []byte(huntText), 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("\n✅ Draft saved → %s\n", absOut)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(huntText)
	fmt.Println(strings.Repeat("─", 60))
	return nil
}
