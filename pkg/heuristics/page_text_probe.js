(sel) => {
    // Returns the human-readable visible text of the page, case-preserved and
    // shadow-DOM aware. Unlike the visible-text presence probe, this is meant
    // for an agent (or LLM) to READ — so it keeps original casing and joins
    // shadow content rather than flattening everything for matching.
    //
    // sel is an optional CSS selector scoping extraction to one region; when
    // empty or not found, the whole document body is used.
    sel = sel || "";

    let root = document.body;
    if (sel) {
        try {
            const scoped = document.querySelector(sel);
            if (scoped) root = scoped;
        } catch (_) { /* invalid selector → fall back to body */ }
    }
    if (!root) return "";

    let t = root.innerText || "";
    // Append shadow-root text the flat innerText misses.
    root.querySelectorAll('*').forEach(el => {
        if (el.shadowRoot) {
            const shadow = Array.from(el.shadowRoot.querySelectorAll('*'))
                .map(e => (e.innerText || "")).join(' ');
            if (shadow) t += "\n" + shadow;
        }
    });
    return t;
}
