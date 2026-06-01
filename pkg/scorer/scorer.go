// Package scorer implements ManulHeart's deterministic multi-signal element scoring.
//
// The primary entry points are:
//   - Score(query, typeHint, mode, el, anchor) → explain.ScoreBreakdown
//   - Rank(query, typeHint, mode, elements, topN, anchor) → []RankedCandidate
//
// Scoring is pure, stateless, and deterministic: same inputs always produce the
// same output. No randomness, no LLM calls.
package scorer

import (
	"math"
	"sort"
	"strings"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
	"github.com/alexbeatnik/ManulHeart/pkg/explain"
)

// ── Public types ──────────────────────────────────────────────────────────────

// AnchorContext holds the resolved context for a NEAR qualifier.
// When non-nil, proximity and attribute-affinity signals are activated.
type AnchorContext struct {
	// Rect is the bounding box of the anchor element.
	Rect dom.Rect
	// XPath is the XPath of the anchor element (for DOM ancestry affinity).
	XPath string
	// FrameIndex is the frame the anchor belongs to.
	FrameIndex int
	// Words are the significant words extracted from the anchor's visible text,
	// used to match candidate attributes like id/class/data-qa.
	Words []string
}

// WeightsConfig holds the high-level signal category weights exposed for
// testing and observability. The internal scorer uses finer-grained per-signal
// weights, but these top-level values define the ordering invariant:
//
//	Cache > Semantic > Text > Attributes > Proximity
type WeightsConfig struct {
	// Cache is the weight for semantic cache and blind context reuse signals.
	Cache float64
	// Semantic is the weight for tag-semantics and role alignment signals.
	Semantic float64
	// Text is the combined weight for visible-text, aria-label, and label signals.
	Text float64
	// Attributes is the weight for HTML id, data-qa, data-testid, class-name,
	// and anchor-attribute-affinity signals.
	Attributes float64
	// Proximity is the weight for NEAR-qualifier spatial scoring.
	Proximity float64
}

// Weights is the package-level scoring weight configuration.
// Tests may read this to verify the calibrated ordering invariants hold:
//
//	Weights.Cache (2.00) > Weights.Semantic (0.60) > Weights.Text (0.45) > Weights.Attributes (0.25) > Weights.Proximity (0.10)
var Weights = WeightsConfig{
	Cache:      2.00,
	Semantic:   0.60,
	Text:       0.45,
	Attributes: 0.25,
	Proximity:  0.10,
}

// RankedCandidate is a scored element with its full explain breakdown.
type RankedCandidate struct {
	// Element is the DOM snapshot of this candidate.
	Element dom.ElementSnapshot
	// Explain holds the full scoring breakdown (scores, signals, rank, chosen).
	Explain explain.Candidate
}

// ── Primary API ───────────────────────────────────────────────────────────────

// Score computes the full scoring breakdown for a single element.
// query is the normalized lowercased target text from the DSL.
// typeHint is the element type keyword from the DSL (button, link, field, …).
// mode is the interaction mode: "clickable", "input", "checkbox", "select", "none".
// anchor is optional; when non-nil, proximity and attribute-affinity are scored.
func Score(query, typeHint, mode string, el *dom.ElementSnapshot, anchor *AnchorContext) explain.ScoreBreakdown {
	if el.IsDisabled {
		return explain.ScoreBreakdown{Total: 0.0, InteractabilityScore: 0.0}
	}

	q := norm(query)

	// ── Text signals ──────────────────────────────────────────────────────────
	exactText := scoreExactText(q, el)
	normText := scoreNormText(q, el)
	labelMatch := scoreLabel(q, el)
	placeholder := scorePlaceholder(q, el)
	aria := scoreAria(q, el)
	dataQA := scoreDataQA(q, el)
	htmlID := scoreID(q, el)

	// ── Structural signals ────────────────────────────────────────────────────
	tagSem := scoreTagSemantics(mode, el)
	typeHintScore := scoreTypeHint(typeHint, el)
	className := scoreClassName(q, el)

	// ── Visibility / interactability ─────────────────────────────────────────
	vis := 1.0
	if !el.IsVisible || el.IsHidden {
		vis = 0.1
	}
	interact := 1.0
	if el.IsDisabled {
		interact = 0.0
	}

	// ── Proximity (NEAR qualifier) ────────────────────────────────────────────
	proximity := 0.0
	anchorAttr := 0.0
	if anchor != nil {
		proximity = scoreNear(el, anchor)
		anchorAttr = scoreAnchorAttrAffinity(anchor, el)
	}

	// ── Category scores (aligned with Python ManulEngine) ─────────────────────
	// Text category: exactText, normText, label, placeholder, aria, dataQA
	textCat := exactText + normText + labelMatch + placeholder + aria + dataQA

	// Attributes category: htmlID, className, anchorAttr
	attrCat := htmlID + className + anchorAttr

	// Semantics category: tagSem, typeHint, modeSynergy, crossModePenalty
	semCat := tagSem + typeHintScore

	isPerfect := exactText == 1.0 || aria == 1.0 || dataQA == 1.0 || labelMatch >= 0.8 || placeholder >= 0.7
	tag := strings.ToLower(el.Tag)
	role := strings.ToLower(el.Role)
	isRealButton := tag == "button" || (tag == "input" && (el.InputType == "submit" || el.InputType == "button" || el.InputType == "image" || el.InputType == "reset")) || role == "button"
	isRealLink := tag == "a" || role == "link"
	isRealInput := (tag == "input" || tag == "textarea") && el.InputType != "submit" && el.InputType != "button" && el.InputType != "image" && el.InputType != "reset" && el.InputType != "radio" && el.InputType != "checkbox" || role == "textbox" || role == "searchbox" || role == "spinbutton" || role == "slider" || el.IsEditable
	isRealCheckbox := (tag == "input" && el.InputType == "checkbox") || role == "checkbox"
	isRealRadio := (tag == "input" && el.InputType == "radio") || role == "radio"

	if isPerfect {
		switch mode {
		case "clickable", "hover":
			if isRealButton || isRealLink || role == "menuitem" || role == "tab" || role == "switch" {
				semCat += 0.5
			}
		case "input":
			if isRealInput {
				semCat += 0.5
			}
		case "select":
			if tag == "select" || tag == "option" || role == "option" || role == "menuitem" || role == "combobox" || role == "button" || tag == "li" {
				semCat += 0.5
			}
		}
	}

	switch mode {
	case "select":
		if isRealCheckbox {
			semCat -= 1.0
		}
		if isRealRadio {
			semCat -= 1.0
		}
	case "input":
		if isRealCheckbox || isRealRadio {
			semCat -= 1.0
		}
	case "clickable":
		if isRealInput && !isRealButton && typeHint == "button" {
			semCat -= 1.0
		}
	}

	// Proximity category
	proxCat := proximity

	// Cache category (not yet implemented in Go; placeholder for parity)
	cacheCat := 0.0

	// ── Weighted total ────────────────────────────────────────────────────────
	penalty := vis * interact
	weightedSum := textCat*Weights.Text +
		attrCat*Weights.Attributes +
		semCat*Weights.Semantic +
		proxCat*Weights.Proximity +
		cacheCat*Weights.Cache

	rawPenalized := weightedSum * penalty
	total := clamp(rawPenalized, 0, 1)

	bd := explain.ScoreBreakdown{
		ExactTextMatch:       exactText,
		NormalizedTextMatch:  normText,
		LabelMatch:           labelMatch,
		PlaceholderMatch:     placeholder,
		AriaMatch:            aria,
		DataQAMatch:          dataQA,
		IDMatch:              htmlID,
		TagSemantics:         tagSem,
		TypeHintAlignment:    typeHintScore,
		VisibilityScore:      vis,
		InteractabilityScore: interact,
		ProximityScore:       proximity,
		RawScore:             rawPenalized,
		Total:                total,
	}
	return bd
}

// Rank scores all elements and returns up to topN candidates sorted by total
// score descending. The first element in the returned slice has Chosen=true.
// anchor is optional; pass nil when no NEAR qualifier is active.
func Rank(query, typeHint, mode string, elements []dom.ElementSnapshot, topN int, anchor *AnchorContext) []RankedCandidate {
	q := norm(query)

	type scored struct {
		elem dom.ElementSnapshot
		bd   explain.ScoreBreakdown
		idx  int // original DOM position for stable tie-breaking
	}

	all := make([]scored, 0, len(elements))
	for i := range elements {
		el := &elements[i]
		bd := Score(q, typeHint, mode, el, anchor)
		all = append(all, scored{elem: *el, bd: bd, idx: i})
	}

	// Sort: highest raw score first; DOM order as deterministic tie-breaker.
	// RawScore is unclamped so that small differences in proximity/semantics
	// are preserved even when Total confidence hits 1.0.
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].bd.RawScore != all[j].bd.RawScore {
			return all[i].bd.RawScore > all[j].bd.RawScore
		}
		return all[i].idx < all[j].idx // earlier in DOM wins on tie
	})

	if topN > 0 && len(all) > topN {
		all = all[:topN]
	}

	result := make([]RankedCandidate, 0, len(all))
	for rank, s := range all {
		sigs := buildSignals(s.bd)
		c := explain.Candidate{
			Rank:        rank + 1,
			XPath:       s.elem.XPath,
			Tag:         s.elem.Tag,
			Role:        s.elem.Role,
			VisibleText: s.elem.VisibleText,
			AriaLabel:   s.elem.AriaLabel,
			Placeholder: s.elem.Placeholder,
			DataQA:      s.elem.DataQA,
			ID:          s.elem.HTMLId,
			IsVisible:   s.elem.IsVisible,
			IsEnabled:   !s.elem.IsDisabled,
			IsEditable:  s.elem.IsEditable,
			Score:       s.bd,
			Signals:     sigs,
			Chosen:      rank == 0,
		}
		result = append(result, RankedCandidate{Element: s.elem, Explain: c})
	}
	return result
}

// ── Scoring signal functions ──────────────────────────────────────────────────

// scoreExactText returns 1.0 for an exact normalized text match, 0 otherwise.
func scoreExactText(q string, el *dom.ElementSnapshot) float64 {
	if q == "" {
		return 0.0
	}
	for _, s := range el.AllTextSignals() {
		if s == q {
			return 1.0
		}
	}
	return 0.0
}

// scoreNormText returns a partial score for substring or word-overlap matches
// across all text signals of the element.
func scoreNormText(q string, el *dom.ElementSnapshot) float64 {
	if q == "" {
		return 0.0
	}
	best := 0.0
	for _, s := range el.AllTextSignals() {
		if s == "" {
			continue
		}
		sc := partialMatch(q, s)
		if sc > best {
			best = sc
		}
	}
	return best
}

// scoreLabel returns how well the element's associated <label> text matches.
func scoreLabel(q string, el *dom.ElementSnapshot) float64 {
	if el.NormLabelText == "" {
		return 0.0
	}
	if el.NormLabelText == q {
		return 0.8
	}
	if strings.Contains(el.NormLabelText, q) {
		return 0.5
	}
	// Word-overlap for multi-word queries like "CPU of Chrome".
	if sigWords := SignificantWords(q); len(sigWords) >= 2 {
		hits := 0
		for _, w := range sigWords {
			if strings.Contains(el.NormLabelText, w) {
				hits++
			}
		}
		coverage := float64(hits) / float64(len(sigWords))
		if coverage >= 1.0 {
			return 0.6
		}
		if coverage > 0.5 {
			return 0.3
		}
	}
	return 0.0
}

// scorePlaceholder returns how well the element's placeholder attribute matches.
func scorePlaceholder(q string, el *dom.ElementSnapshot) float64 {
	if el.NormPlaceholder == "" {
		return 0.0
	}
	if el.NormPlaceholder == q {
		return 0.7
	}
	if strings.Contains(el.NormPlaceholder, q) {
		return 0.4
	}
	return 0.0
}

// scoreAria returns how well the element's aria-label matches.
func scoreAria(q string, el *dom.ElementSnapshot) float64 {
	if el.NormAriaLabel == "" {
		return 0.0
	}
	if el.NormAriaLabel == q {
		return 1.0
	}
	if strings.Contains(el.NormAriaLabel, q) {
		return 0.55
	}
	return 0.0
}

// scoreDataQA returns how well the element's data-qa / data-testid matches.
func scoreDataQA(q string, el *dom.ElementSnapshot) float64 {
	if el.NormDataQA == "" {
		return 0.0
	}
	if el.NormDataQA == q {
		return 1.0
	}
	if strings.Contains(el.NormDataQA, q) {
		return 0.5
	}
	return 0.0
}

// scoreID returns how well the element's html id attribute matches.
func scoreID(q string, el *dom.ElementSnapshot) float64 {
	if el.NormHTMLId == "" {
		return 0.0
	}
	normalized := el.NormHTMLId
	variants := []string{
		q,
		strings.ReplaceAll(q, " ", "-"),
		strings.ReplaceAll(q, " ", "_"),
		strings.ReplaceAll(q, " ", ""),
	}
	for _, v := range variants {
		if normalized == v {
			return 0.7
		}
	}
	if strings.Contains(normalized, strings.ReplaceAll(q, " ", "")) {
		return 0.3
	}
	return 0.0
}

// scoreTagSemantics returns a score for how well the element's tag/role aligns
// with the expected interaction mode.
//
// Values are calibrated so that the weighted contribution (score*Weights.Semantic)
// matches the effective contribution under the old signal-level weighting.
func scoreTagSemantics(mode string, el *dom.ElementSnapshot) float64 {
	tag := el.Tag
	role := strings.ToLower(el.Role)

	switch mode {
	case "none", "locate":
		if tag == "td" || tag == "th" || tag == "li" || tag == "span" ||
			tag == "p" || tag == "dd" || tag == "dt" || tag == "figcaption" || tag == "caption" {
			return 0.025
		}
		if len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6' {
			return 0.03
		}
		if tag == "div" || tag == "section" || tag == "article" {
			return 0.02
		}
		if tag == "option" {
			return 0.02
		}
		if tag == "button" || tag == "a" || tag == "input" || tag == "select" {
			return 0.015
		}
		return 0.01

	case "input":
		if tag == "input" || tag == "textarea" {
			if el.InputType == "password" {
				return 0.07
			}
			return 0.065
		}
		if role == "textbox" || role == "spinbutton" || role == "combobox" {
			return 0.05
		}
		if el.IsEditable {
			return 0.04
		}
		// Strict penalty for non-inputs in input mode
		return -1.0

	case "checkbox":
		if tag == "input" && (el.InputType == "checkbox" || el.InputType == "radio") {
			return 0.06
		}
		if role == "checkbox" || role == "radio" || role == "switch" {
			return 0.05
		}
		// Strict penalty for non-checkbox elements in checkbox mode.
		return -1.0

	case "select":
		if tag == "select" {
			return 0.06
		}
		if role == "listbox" || role == "combobox" {
			return 0.05
		}
		// Strict penalty for non-selects in select mode
		return -1.0

	default: // clickable
		if tag == "button" || tag == "a" || tag == "summary" {
			return 0.055
		}
		if tag == "input" && (el.InputType == "checkbox" || el.InputType == "radio") {
			return 0.04
		}
		if role == "button" || role == "link" || role == "menuitem" || role == "tab" {
			return 0.045
		}
		if role == "checkbox" || role == "radio" || role == "switch" {
			return 0.04
		}
		if tag == "input" && (el.InputType == "submit" || el.InputType == "button") {
			return 0.045
		}
		if tag == "label" {
			return 0.025
		}
		return 0.01
	}
}

// scoreTypeHint returns a score for how well the element matches the explicit
// type hint extracted from the DSL command ("button", "link", "field", …).
func scoreTypeHint(hint string, el *dom.ElementSnapshot) float64 {
	if hint == "" {
		return 0.0
	}
	tag := el.Tag
	role := strings.ToLower(el.Role)
	switch hint {
	case "button":
		if tag == "button" || (tag == "input" && (el.InputType == "submit" || el.InputType == "button")) {
			return 0.45
		}
		if role == "button" {
			return 0.4
		}
	case "link":
		if tag == "a" || role == "link" {
			return 0.4
		}
	case "field", "input", "textarea":
		if tag == "input" || tag == "textarea" || el.IsEditable {
			return 0.4
		}
	case "checkbox":
		if tag == "input" && el.InputType == "checkbox" {
			return 0.7
		}
		if role == "checkbox" {
			return 0.7
		}
		return -0.6
	case "radio":
		if tag == "input" && el.InputType == "radio" {
			return 0.7
		}
		if role == "radio" {
			return 0.7
		}
		return -0.6
	case "dropdown", "select":
		if tag == "select" || role == "listbox" || role == "combobox" {
			return 0.4
		}
	case "element":
		return 0.05 // generic hint — minimal signal
	}
	return 0.0
}

// scoreNear returns a [0.0, 1.0] proximity score for a NEAR qualifier.
// Uses linear spatial decay blended with DOM ancestry affinity:
//
//	score = spatial*0.45 + domAffinity*0.55
//
// This helps card/list layouts prefer the button in the same product card
// over a slightly closer button in an adjacent card.
func scoreNear(el *dom.ElementSnapshot, anchor *AnchorContext) float64 {
	if el.FrameIndex != anchor.FrameIndex {
		return 0.0
	}
	const threshold = 500.0
	cx := el.Rect.Left + el.Rect.Width/2
	cy := el.Rect.Top + el.Rect.Height/2
	ax := anchor.Rect.Left + anchor.Rect.Width/2
	ay := anchor.Rect.Top + anchor.Rect.Height/2
	dist := math.Sqrt((cx-ax)*(cx-ax) + (cy-ay)*(cy-ay))
	if dist > threshold {
		return 0.0
	}
	spatialScore := 1.0 - dist/threshold

	if anchor.XPath == "" || el.XPath == "" {
		return spatialScore
	}

	anchorParts := xpathParts(anchor.XPath)
	candidateParts := xpathParts(el.XPath)
	commonDepth := 0
	for i := 0; i < len(anchorParts) && i < len(candidateParts); i++ {
		if anchorParts[i] == candidateParts[i] {
			commonDepth++
		} else {
			break
		}
	}
	maxLen := len(anchorParts)
	if len(candidateParts) > maxLen {
		maxLen = len(candidateParts)
	}
	if maxLen == 0 {
		return spatialScore
	}
	domAffinity := float64(commonDepth) / float64(maxLen)
	return math.Min(1.0, spatialScore*0.45+domAffinity*0.55)
}

// scoreAnchorAttrAffinity rewards candidates whose dev-facing attributes
// (id, class, data-qa) contain words from the NEAR anchor text.
// Matches ManulEngine: product cards encode the item label in button IDs
// like add-to-cart-sauce-labs-fleece-jacket.
func scoreAnchorAttrAffinity(anchor *AnchorContext, el *dom.ElementSnapshot) float64 {
	if len(anchor.Words) == 0 {
		return 0.0
	}
	replacer := strings.NewReplacer("-", " ", "_", " ")
	devText := replacer.Replace(strings.ToLower(
		el.NormHTMLId + " " + el.ClassName + " " + el.NormDataQA + " " + el.DataTestID,
	))
	hits := 0
	for _, w := range anchor.Words {
		if strings.Contains(devText, w) {
			hits++
		}
	}
	if hits == 0 {
		return 0.0
	}
	coverage := float64(hits) / float64(len(anchor.Words))
	switch {
	case coverage >= 1.0:
		return 0.6
	case coverage >= 0.75:
		return 0.35
	case coverage >= 0.5:
		return 0.12
	default:
		return 0.0
	}
}

// scoreClassName computes a word-overlap score between the query and the
// element's CSS class names.
func scoreClassName(q string, el *dom.ElementSnapshot) float64 {
	if el.ClassName == "" || q == "" {
		return 0.0
	}
	replacer := strings.NewReplacer("-", " ", "_", " ")
	clsNorm := replacer.Replace(strings.ToLower(el.ClassName))
	hits := 0
	for _, w := range strings.Fields(q) {
		if len(w) >= 3 && strings.Contains(clsNorm, w) {
			hits++
		}
	}
	if hits == 0 {
		return 0.0
	}
	return math.Min(float64(hits)*0.08, 0.4)
}

// ── Utilities ─────────────────────────────────────────────────────────────────

// partialMatch returns a [0, 1] score for a substring or word-overlap match
// between query q and candidate text s (both pre-normalized).
//
// It does NOT do reverse-containment: only s.contains(q), not q.contains(s).
// Reverse-containment was removed because it caused short elements (e.g. "Update")
// to score the same as longer ones (e.g. "Update Profile") for query "Update Profile".
func partialMatch(q, s string) float64 {
	if q == "" || s == "" {
		return 0.0
	}
	if s == q {
		return 1.0
	}
	score := 0.0
	// Forward containment only: element text contains the query.
	if strings.Contains(s, q) {
		// Use rune count instead of byte len to handle emojis properly
		qLen := float64(len([]rune(q)))
		sLen := float64(len([]rune(s)))
		score = clamp(qLen/sLen, 0.4, 0.95)
	}
	// Word-overlap with stop-word filtering and minimum word length.
	overlap := wordOverlap(q, s)
	if overlap > score {
		return overlap
	}
	return score
}

// wordOverlap returns the fraction of significant query words found in s.
// Minimum word length ≥ 2 and stop-word filtering prevent single-char false positives.
func wordOverlap(q, s string) float64 {
	qWords := SignificantWords(q)
	if len(qWords) == 0 {
		return 0.0
	}
	hits := 0
	for _, w := range qWords {
		if strings.Contains(s, w) {
			hits++
		}
	}
	return clamp(float64(hits)/float64(len(qWords)), 0, 1)
}

// xpathParts splits an XPath string into its non-empty path components.
func xpathParts(xpath string) []string {
	var parts []string
	for _, p := range strings.Split(xpath, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// stopWords are common English function words excluded from word-overlap matching.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "of": true, "in": true,
	"for": true, "to": true, "is": true, "on": true, "at": true,
	"by": true, "or": true, "and": true, "with": true, "from": true,
	"that": true, "this": true, "it": true, "its": true, "be": true,
	"as": true, "are": true, "was": true, "were": true, "not": true,
	"all": true,
}

// SignificantWords returns query words with length ≥ 2 and not a stop word.
func SignificantWords(s string) []string {
	var words []string
	for _, w := range strings.Fields(s) {
		if len(w) >= 2 && !stopWords[w] {
			words = append(words, w)
		}
	}
	return words
}

// norm lowercases and trims a string (consistent with dom.ElementSnapshot.Normalize).
func norm(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// clamp constrains v to [min, max].
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// buildSignals converts a ScoreBreakdown into the human-readable signal list
// used by --explain mode and the JSON output.
func buildSignals(bd explain.ScoreBreakdown) []explain.CandidateSignal {
	var signals []explain.CandidateSignal
	add := func(name string, score float64) {
		if score > 0 {
			signals = append(signals, explain.CandidateSignal{Signal: name, Score: score})
		}
	}
	add("exact_text", bd.ExactTextMatch)
	add("normalized_text", bd.NormalizedTextMatch)
	add("label", bd.LabelMatch)
	add("placeholder", bd.PlaceholderMatch)
	add("aria_label", bd.AriaMatch)
	add("data_qa", bd.DataQAMatch)
	add("html_id", bd.IDMatch)
	add("tag_semantics", bd.TagSemantics)
	add("type_hint", bd.TypeHintAlignment)
	add("proximity", bd.ProximityScore)
	return signals
}

// ── Legacy compatibility shims ────────────────────────────────────────────────
// These keep any callers of the old API compiling.

// ScoreCandidate is the legacy scoring entry point kept for backward compatibility.
// New code should call Score() directly.
func ScoreCandidate(elem dom.ElementSnapshot, intent Intent, _ LegacyWeights, _ map[string]dom.ElementSnapshot) explain.CandidateResult {
	bd := Score(intent.Text, intent.Role, "clickable", &elem, nil)
	return explain.CandidateResult{
		NodeID:      elem.HTMLId,
		TagName:     elem.Tag,
		TextContent: elem.VisibleText,
		RawScore:    bd.RawScore,
		Score:       bd.Total,
		Signals:     nil,
	}
}

// RankCandidates is the legacy ranking entry point kept for backward compatibility.
func RankCandidates(elements []dom.ElementSnapshot, intent Intent, _ LegacyWeights) []explain.CandidateResult {
	ranked := Rank(intent.Text, intent.Role, "clickable", elements, 0, nil)
	out := make([]explain.CandidateResult, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, explain.CandidateResult{
			NodeID:      r.Element.HTMLId,
			TagName:     r.Element.Tag,
			TextContent: r.Element.VisibleText,
			RawScore:    r.Explain.Score.RawScore,
			Score:       r.Explain.Score.Total,
			Signals:     nil,
		})
	}
	return out
}

// Intent is the legacy targeting intent type.
type Intent struct {
	Text             string
	Role             string
	NearAnchorNodeID string
}

// LegacyWeights is the legacy signal-weight map type.
// New code should use WeightsConfig / the Weights package variable instead.
type LegacyWeights map[Signal]float64

// Signal is a named scoring channel.
type Signal string

// DefaultWeights returns the default signal weights (legacy API).
func DefaultWeights() LegacyWeights { return LegacyWeights{} }
