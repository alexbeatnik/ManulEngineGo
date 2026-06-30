# Loops & Page Objects

> ManulEngine (Go) now supports the same loop constructs and page-name registry as Python ManulEngine.

## Loops

### `REPEAT N TIMES:`

Execute a block of commands a fixed number of times.

```hunt
REPEAT 3 TIMES:
    CLICK the 'Next' button
    VERIFY that 'Step {i}' is present
```

The loop counter is automatically available as `{i}` (0-based).

### `FOR EACH {var} IN {collection}:`

Iterate over a comma-separated list stored in a variable.

```hunt
@var: {products} = Laptop, Headphones, Mouse

FOR EACH {product} IN {products}:
    FILL 'Search' field with '{product}'
    CLICK the 'Add to Cart' button NEAR '{product}'
```

### `WHILE condition:`

Repeat while a condition is true (max 100 iterations for safety).

```hunt
WHILE button 'Load More' exists:
    CLICK the 'Load More' button
    WAIT 1
```

### Nested loops

Loops can be nested inside each other and inside `IF` blocks:

```hunt
REPEAT 2 TIMES:
    FOR EACH {item} IN {items}:
        IF {item} != 'Skip':
            CLICK '{item}' button
```

---

## Page Objects (pages/ directory)

ManulEngine (Go) uses a `pages/` directory next to your hunt files to map URLs to human-readable page names.

### Fragment format

Each site gets its own JSON file: `pages/<safe-netloc>.json`

**Lean form** (recommended):
```json
{
    "site": "https://example.com/",
    "Domain": "Example",
    "https://example.com/login": "Login Page",
    "https://example.com/dashboard": "Dashboard"
}
```

**Wrapped form** (backward-compatible):
```json
{
    "https://example.com/": {
        "Domain": "Example",
        "/login": "Login Page"
    }
}
```

### Matching logic

1. **Longest-prefix site match** — `https://app.example.com/` shadows `https://example.com/`
2. **Exact URL match**
3. **Regex pattern** via `regexp.MatchString`
4. **Substring fallback**
5. **`"Domain"` key** as final fallback

### Auto-populate

Unknown URLs are automatically added as `Auto: domain/path` placeholders:

```json
{
    "site": "https://new-site.io/",
    "Domain": "Auto: new-site.io",
    "https://new-site.io/profile": "Auto: new-site.io/profile"
}
```

This happens transparently on the first `NAVIGATE` to an unmapped URL.

### Using page names in custom controls

Custom controls can target specific pages:

```go
runtime.RegisterCustomControl("Checkout Page", "React Datepicker", handler)
```

The page label is resolved via `document.title` first, then the `pages/` registry, then URL-derived fallback.

### Comparison with Python ManulEngine

| Feature | Python ManulEngine | Go ManulEngine (Go) |
|---------|-------------------|---------------|
| Loop types | REPEAT, FOR EACH, WHILE | REPEAT, FOR EACH, WHILE |
| Max WHILE iterations | `MAX_LOOP_ITERATIONS = 100` | hard limit = 100 |
| Loop variable | `{i}` auto-set for REPEAT | `{i}` auto-set for REPEAT |
| Page registry | `pages/<site>.json` | `pages/<site>.json` |
| Auto-populate | `_auto_populate_registry()` | `Registry.autoPopulate()` |
| Registry reload | every `lookup_page_name()` call | every `LookupPageName()` call |
| Lean/wrapped forms | both supported | both supported |
