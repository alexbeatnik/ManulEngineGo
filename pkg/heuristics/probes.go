// Package heuristics provides the in-page JavaScript probes that ManulEngine (Go)
// injects into the browser to collect normalized candidate data.
//
// Probes are first-class components of the engine targeting pipeline.
// They are invoked on every target-based DSL command as the primary DOM
// interrogation step.
//
// JS probes are embedded from .js files so they benefit from syntax
// highlighting and can be edited without touching Go source.
package heuristics

import (
	"encoding/json"
	"fmt"

	_ "embed"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

// snapshotProbeJS is the primary DOM snapshot probe.
// It traverses the entire visible DOM (including shadow roots) and returns
// a JSON-serializable snapshot of all candidate elements.
//
//go:embed snapshot_probe.js
var snapshotProbeJS string

// visibleTextProbeJS quickly collects visible page text without full traversal.
//
//go:embed visible_text_probe.js
var visibleTextProbeJS string

// xpathResolveProbeJS resolves an XPath and returns its current visibility state.
//
//go:embed xpath_probe.js
var xpathResolveProbeJS string

// extractDataProbeJS is the dedicated data-extraction probe for EXTRACT commands.
//
//go:embed extract_data.js
var extractDataProbeJS string

// pageTextProbeJS reads the case-preserved, shadow-DOM-aware visible text of
// the page (or a selector-scoped region) for an agent/LLM to consume.
//
//go:embed page_text_probe.js
var pageTextProbeJS string

// BuildSnapshotProbe returns the snapshot probe JS string.
// This is the primary probe used by every targeting command.
func BuildSnapshotProbe() string {
	return snapshotProbeJS
}

// BuildVisibleTextProbe returns the lightweight visible-text probe JS string.
// Used by VERIFY commands to check text presence without full element extraction.
func BuildVisibleTextProbe() string {
	return visibleTextProbeJS
}

// BuildXPathProbe returns the XPath-resolve probe JS string.
// Used to verify element state (visibility, disabled) after targeting.
func BuildXPathProbe() string {
	return xpathResolveProbeJS
}

// BuildExtractProbe returns the data-extraction probe JS string.
// Used by EXTRACT commands to pull values from tables and text nodes.
func BuildExtractProbe() string {
	return extractDataProbeJS
}

// BuildPageTextProbe returns the page-text probe JS string. It takes an
// optional CSS selector argument (empty → whole body) and returns the
// case-preserved visible text, for agent/LLM consumption rather than matching.
func BuildPageTextProbe() string {
	return pageTextProbeJS
}

// BuildProbeScript returns the snapshot probe JS for backward compatibility.
// New code should use BuildSnapshotProbe().
func BuildProbeScript() string {
	return snapshotProbeJS
}

// ParseProbeResult parses the raw JSON result of a DOM probe into a slice of
// ElementSnapshot values. The raw argument may be a string, []byte, or any
// JSON-serializable value (e.g. the return value of cdp.Evaluate or cdp.CallFunctionOn).
func ParseProbeResult(raw interface{}) ([]dom.ElementSnapshot, error) {
	if raw == nil {
		return nil, nil
	}

	var payload []byte
	switch value := raw.(type) {
	case string:
		payload = []byte(value)
	case []byte:
		payload = value
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal probe result: %w", err)
		}
		payload = encoded
	}

	var wrap struct {
		URL         string                `json:"url"`
		Title       string                `json:"title"`
		VisibleText string                `json:"visible_text"`
		Elements    []dom.ElementSnapshot `json:"elements"`
	}
	if err := json.Unmarshal(payload, &wrap); err != nil {
		// Fallback for older probes that returned raw arrays
		var fallback []dom.ElementSnapshot
		if err2 := json.Unmarshal(payload, &fallback); err2 != nil {
			return nil, fmt.Errorf("unmarshal probe result: %w", err)
		}
		wrap.Elements = fallback
	}

	elements := wrap.Elements

	// Populate normalized fields for scoring.
	for i := range elements {
		elements[i].Normalize()
	}
	return elements, nil
}
