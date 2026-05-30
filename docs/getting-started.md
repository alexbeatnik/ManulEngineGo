# Getting Started with ManulHeart

## Prerequisites

- Go 1.26+ (required by module)
- Google Chrome (or Chromium) installed
- A terminal

---

## 1. Build

```bash
git clone https://github.com/alexbeatnik/ManulHeart.git
cd ManulHeart
go build -o manul ./cmd/manul
```

---

## 2. Run an example (auto-launches Chrome)

```bash
manul examples/login.hunt
```

Chrome is launched automatically with remote debugging, the hunt is executed,
and Chrome is closed when done. No manual browser setup required.

**Headless (no window):**
```bash
manul examples/login.hunt --headless
```

**Connect to existing Chrome** (if you already have Chrome running):
```bash
manul examples/login.hunt --cdp http://127.0.0.1:9222
```

---

## 3. Write a .hunt file

Create `tests/my_flow.hunt`:

```
@context: Login to the demo application and verify the secure area
@title: Demo Login

STEP 1: Navigate and verify start page
    NAVIGATE to 'https://the-internet.herokuapp.com/login'
    VERIFY that 'Login Page' is present

STEP 2: Submit credentials
    Fill 'Username' field with 'tomsmith'
    Fill 'Password' field with 'SuperSecretPassword!'
    Click the 'Login' button

STEP 3: Verify success
    VERIFY that 'You logged into a secure area!' is present

DONE.
```

**DSL rules:**
- Commands are case-insensitive (`NAVIGATE`, `navigate`, and `Navigate` all work).
- Target text goes inside single quotes: `Click the 'Login' button`.
- The element type hint (`button`, `link`, `field`, etc.) after the target is optional
  but improves scoring accuracy.
- `STEP N:` blocks are structural labels and do not affect execution order.
- `DONE.` terminates the file; lines after it are ignored.

---

## 4. Run the hunt file

```bash
manul tests/my_flow.hunt
```

Expected output:
```
Launching Chrome (port 9222, profile /tmp/manulheart-chrome)…
ManulHeart — tests/my_flow.hunt
Title: Demo Login
Commands: 6
CDP: http://127.0.0.1:9222

[📦 BLOCK START] STEP 1: Navigate and verify start page
  [▶️  ACTION START] NAVIGATE to 'https://the-internet.herokuapp.com/login'
  [✅ ACTION PASS] duration: 0.74s
  [▶️  ACTION START] VERIFY that 'Login Page' is present
  [✅ ACTION PASS] duration: 0.06s
...
✓ All 6 steps passed (2330ms)
```

---

## 5. Run a single step interactively

```bash
manul run-step "NAVIGATE to 'https://example.com'"
manul run-step "Click the 'More information...' link"
manul run-step "VERIFY that 'IANA' is present"
```

Connect to an already-running Chrome instead of auto-launching:
```bash
manul run-step "Click the 'Login' button" --cdp http://127.0.0.1:9222
```

---

## 6. JSON output

Both `run` and `run-step` support `--json` for machine-readable output:

```bash
manul examples/login.hunt --json 2>/dev/null | jq '.results[] | {step:.step, success:.success, winner:.winner_xpath, score:.winner_score}'
```

The JSON result includes:
- Per-command `ExecutionResult` with ranked candidates and score breakdowns
- `winner_xpath`, `winner_score` for the resolved element
- `candidates_considered`, `ranked_candidates` with full `ScoreBreakdown`
- `duration_ms`, `success`, `error`

---

## 7. Verbose / debug mode

```bash
manul examples/login.hunt --verbose
```

Verbose mode logs:
- Number of candidates discovered per targeting call
- The best candidate's visible text, XPath, and score
- Scroll and action dispatch details

---

## 8. Pipe a hunt from stdin (`0.0.1.4`+)

Pass `-` as the target to read a hunt script from stdin instead of a file:

```bash
cat examples/login.hunt | manul -
echo 'STEP 1: Open page
    NAVIGATE to "https://example.com"' | manul - --headless
```

`@import:` paths resolve against the **current working directory**. On failure, ManulHeart still emits a partial `*HuntResult` with per-step errors — useful for editor integrations and CI pipelines that need more detail than `exit 1`.

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `connect to browser: dial …: connection refused` | If using `--cdp`, ensure Chrome is running with `--remote-debugging-port=9222`. Without `--cdp`, the driver auto-launches Chrome. |
| `no page target found` | The Chrome instance has no open tab; open one manually or let the driver auto-launch. |
| `best score 0.xxx below threshold 0.15` | The target text doesn't match any element well. Use `--verbose` and `--json` to inspect candidates. |
| `VERIFY: "X" not found on page` | The text is not in the visible DOM within the verify timeout. Check the page state. |
