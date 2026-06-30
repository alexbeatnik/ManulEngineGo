// Package pages implements the ManulEngine (Go) page-name registry.
//
// It mirrors Python ManulEngine's pages/ directory system:
//   - Per-site JSON fragments under <project>/pages/<safe_netloc>.json
//   - Lean shape:   { "site": "https://example.com/", "Domain": "Example", "/login": "Login Page" }
//   - Wrapped shape:{ "https://example.com/": { "Domain": "Example", "/login": "Login Page" } }
//   - Auto-populate unknown URLs as "Auto: domain/path" placeholders.
package pages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Registry maps site_root_url → { pattern_or_"Domain": page_name }.
type Registry struct {
	mu   sync.RWMutex
	data map[string]map[string]string // site_root → {pattern → name}
	dir  string
}

// NewRegistry creates a Registry backed by *dir*.
// If dir is empty it defaults to "pages" under the current working directory.
func NewRegistry(dir string) *Registry {
	if dir == "" {
		cwd, _ := os.Getwd()
		dir = filepath.Join(cwd, "pages")
	}
	r := &Registry{dir: dir, data: map[string]map[string]string{}}
	r.Reload()
	return r
}

// Reload merges every pages/*.json fragment into the in-memory registry.
func (r *Registry) Reload() {
	r.mu.Lock()
	defer r.mu.Unlock()

	merged := map[string]map[string]string{}
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(r.dir, e.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var raw map[string]any
		if jsonErr := json.Unmarshal(data, &raw); jsonErr != nil {
			continue
		}
		for siteKey, siteVal := range normalizeFragment(raw) {
			existing, ok := merged[siteKey]
			if !ok {
				existing = map[string]string{}
				merged[siteKey] = existing
			}
			for k, v := range siteVal {
				existing[k] = v
			}
		}
	}
	r.data = merged
}

// LookupPageName matches *url* against the registry and returns the mapped page name.
//
// Matching logic (same as Python ManulEngine):
//  1. Reload registry from disk on every call so manual edits are picked up.
//  2. Find best-matching site block by longest-prefix against URL scheme+host+path.
//  3. Within site block:
//     a. Exact URL equality.
//     b. Regex via regexp.MatchString (invalid regex falls back to substring).
//     c. The special "Domain" key as fallback.
//  4. If no site block matches, auto-populate a placeholder.
func (r *Registry) LookupPageName(rawURL string) string {
	r.Reload()

	r.mu.RLock()
	defer r.mu.RUnlock()

	// 2. Find best (longest-prefix) site block.
	var bestSite string
	for siteRoot := range r.data {
		if belongsToSite(rawURL, siteRoot) {
			if bestSite == "" || len(siteRoot) > len(bestSite) {
				bestSite = siteRoot
			}
		}
	}

	// 3. Match within site block.
	if bestSite != "" {
		pages := r.data[bestSite]
		domainName := pages["Domain"]

		// a. Exact URL match.
		if name, ok := pages[rawURL]; ok {
			return name
		}

		// b. Regex / substring patterns.
		for pattern, name := range pages {
			if pattern == "Domain" || pattern == "site" {
				continue
			}
			matched, reErr := regexp.MatchString(pattern, rawURL)
			if reErr == nil && matched {
				return name
			}
			if strings.Contains(rawURL, pattern) {
				return name
			}
		}

		// c. Domain fallback.
		if domainName != "" {
			return domainName
		}
	}

	// 4. Auto-populate.
	return r.autoPopulate(rawURL)
}

// AutoPopulate creates a placeholder entry for an unmapped URL and persists it.
func (r *Registry) autoPopulate(rawURL string) string {
	parsed := parseURL(rawURL)

	siteRoot := fmt.Sprintf("%s://%s/", parsed.scheme, parsed.host)
	slug := (parsed.host + parsed.path)
	slug = strings.Trim(slug, "/")
	placeholder := fmt.Sprintf("Auto: %s", slug)
	if slug == "" {
		placeholder = fmt.Sprintf("Auto: %s", rawURL)
	}

	fragmentPath := filepath.Join(r.dir, safeSiteFilename(siteRoot))

	// Read current on-disk fragment.
	siteBlock := map[string]string{}
	fragmentSite := siteRoot
	if data, readErr := os.ReadFile(fragmentPath); readErr == nil {
		var raw map[string]any
		if jsonErr := json.Unmarshal(data, &raw); jsonErr == nil {
			if allDict(raw) && len(raw) > 0 {
				// Wrapped form.
				if _, ok := raw[siteRoot]; ok {
					siteBlock = dictStringString(raw[siteRoot])
				} else {
					for k, v := range raw {
						fragmentSite = k
						siteBlock = dictStringString(v)
						break
					}
				}
			} else {
				// Lean form.
				if s, ok := raw["site"].(string); ok && s != "" {
					fragmentSite = s
				}
				siteBlock = dictStringString(raw)
			}
		}
	}

	// Merge: never overwrite existing keys.
	updated := false
	if _, ok := siteBlock["Domain"]; !ok {
		siteBlock["Domain"] = placeholder
		updated = true
	}
	if _, ok := siteBlock[rawURL]; !ok {
		siteBlock[rawURL] = placeholder
		updated = true
	}

	// Update in-memory registry.
	r.mu.RUnlock()
	r.mu.Lock()
	siteEntry, ok := r.data[fragmentSite]
	if !ok {
		siteEntry = map[string]string{}
		r.data[fragmentSite] = siteEntry
	}
	for k, v := range siteBlock {
		siteEntry[k] = v
	}
	r.mu.Unlock()
	r.mu.RLock()

	// Persist back to disk.
	if updated {
		_ = os.MkdirAll(r.dir, 0755)
		payload := map[string]any{"site": fragmentSite}
		for k, v := range siteBlock {
			payload[k] = v
		}
		if data, marshalErr := json.MarshalIndent(payload, "", "    "); marshalErr == nil {
			_ = os.WriteFile(fragmentPath, append(data, '\n'), 0644)
		}
	}

	return placeholder
}

// Dir returns the directory backing this registry.
func (r *Registry) Dir() string {
	return r.dir
}

// ── internal helpers ──────────────────────────────────────────────────────────

// normalizeFragment turns a single JSON payload into {site_root: {pattern: name}}.
func normalizeFragment(raw map[string]any) map[string]map[string]string {
	out := map[string]map[string]string{}
	if len(raw) == 0 {
		return out
	}

	// Wrapped form: every top-level value is a dict.
	if allDict(raw) {
		for siteKey, siteVal := range raw {
			sk := fmt.Sprint(siteKey)
			if sk == "" {
				continue
			}
			out[sk] = dictStringString(siteVal)
		}
		return out
	}

	// Lean form: site root comes from "site" field.
	site := ""
	if s, ok := raw["site"].(string); ok {
		site = s
	}
	if site == "" {
		return out
	}
	fields := map[string]string{}
	for k, v := range raw {
		if k == "site" {
			continue
		}
		fields[k] = fmt.Sprint(v)
	}
	if len(fields) > 0 {
		out[site] = fields
	}
	return out
}

func allDict(m map[string]any) bool {
	for _, v := range m {
		if _, ok := v.(map[string]any); !ok {
			return false
		}
	}
	return len(m) > 0
}

func dictStringString(v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v2 := range m {
		out[k] = fmt.Sprint(v2)
	}
	return out
}

func belongsToSite(candidateURL, siteRoot string) bool {
	c := parseURL(candidateURL)
	s := parseURL(siteRoot)
	if c.scheme != s.scheme || c.host != s.host {
		return false
	}
	cPath := strings.Trim(c.path, "/")
	sPath := strings.Trim(s.path, "/")
	if sPath == "" {
		return true
	}
	return cPath == sPath || strings.HasPrefix(cPath, sPath+"/")
}

type urlParts struct {
	scheme string
	host   string
	path   string
}

// parseURL is a lightweight URL parser sufficient for page-name matching.
func parseURL(raw string) urlParts {
	// Handle scheme.
	scheme := "https"
	rest := raw
	if idx := strings.Index(rest, "://"); idx >= 0 {
		scheme = strings.ToLower(rest[:idx])
		rest = rest[idx+3:]
	}
	// Split host and path.
	host := rest
	path := ""
	if idx := strings.Index(rest, "/"); idx >= 0 {
		host = rest[:idx]
		path = rest[idx:]
	}
	return urlParts{scheme: scheme, host: host, path: path}
}

func safeSiteFilename(siteRoot string) string {
	p := parseURL(siteRoot)
	netloc := p.host
	if netloc == "" {
		netloc = siteRoot
	}
	re := regexp.MustCompile(`[^0-9A-Za-z._-]`)
	safe := strings.Trim(re.ReplaceAllString(netloc, "_"), "_")
	if safe == "" {
		safe = "site"
	}
	return safe + ".json"
}

// setdefault helper for maps.
type stringMapMap map[string]map[string]string

func (m stringMapMap) setdefault(key string, value map[string]string) map[string]string {
	if existing, ok := m[key]; ok {
		return existing
	}
	m[key] = value
	return value
}

// MigrateLegacyJSON reads a legacy pages.json (wrapped or lean) and writes
// per-site fragments into <outDir>/<safe_netloc>.json.
func MigrateLegacyJSON(legacyPath, outDir string) error {
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return fmt.Errorf("read legacy pages.json: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse legacy pages.json: %w", err)
	}

	fragments := normalizeFragment(raw)
	if len(fragments) == 0 {
		return fmt.Errorf("no site blocks found in %s", legacyPath)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("create pages dir: %w", err)
	}

	for siteRoot, fields := range fragments {
		payload := map[string]any{"site": siteRoot}
		for k, v := range fields {
			payload[k] = v
		}
		data, err := json.MarshalIndent(payload, "", "    ")
		if err != nil {
			continue
		}
		path := filepath.Join(outDir, safeSiteFilename(siteRoot))
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			return fmt.Errorf("write fragment %s: %w", path, err)
		}
	}
	return nil
}
