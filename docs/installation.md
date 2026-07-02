# Installation

> **ManulEngine (Go) 0.1.0**

## Requirements

| Requirement | Version | Notes |
|-------------|---------|-------|
| **Go** | 1.26+ | Build-time only — the artifact is a single static binary |
| **Google Chrome / Chromium** | any recent | Must be on `PATH` (or set `--channel` / `--executable-path`) — the engine drives it directly over CDP |
| **OS** | Linux, macOS, Windows | All three platforms supported |

No Playwright, no Node.js, no Python, no bundled browser download — the only external Go dependency is `gorilla/websocket`.

**Optional:**

| Tool | Purpose |
|------|---------|
| **VS Code** | For the companion Manul Engine Extension (Test Explorer, debug runner) |
| **make** | Convenience build/install targets |

## Build & Install

```bash
git clone https://github.com/alexbeatnik/ManulEngineGo.git
cd ManulEngineGo

make build            # → ./manul
make install          # → ~/.local/bin/manul (user-local)
make install-system   # → /usr/local/bin/manul (system-wide)
```

Or with Go directly:

```bash
go build -o manul ./cmd/manul
```

Or straight from the module:

```bash
go install github.com/alexbeatnik/ManulEngineGo/cmd/manul@latest
```

> Expose the binary as a PATH command named `manul` — editor extensions and automation tooling look it up by that name.

Verify the build:

```bash
go test ./...        # full unit + synthetic suite (no network needed)
```

## Configuration File

Create `manul_engine_configuration.json` in your project root (read from the CWD). All keys are optional:

```json
{
  "browser": "chromium",
  "headless": false
}
```

Layering: **CLI flags → `MANUL_*` env vars → JSON file → defaults**. See [README_DEV.md → Configuration](../README_DEV.md#%EF%B8%8F-configuration-manul_engine_configurationjson) for the full key table.

## VS Code Extension

Install the companion Manul Engine Extension for VS Code from the Marketplace:

```bash
code --install-extension manul-engine.manul-engine
```

Open a workspace containing `go.mod` and the extension auto-detects the Go runtime: `CALL GO` snippets, hook-block scaffolds, gutter breakpoints, the debug overlay, and Test Explorer all work against the `manul` binary.

## Verifying the Installation

```bash
# Check the CLI is available
manul --help
manul --version      # → manul 0.1.0

# Run a quick smoke test
echo '@context: Quick test
@title: Smoke

STEP 1: Open example.com
    NAVIGATE to https://example.com
    VERIFY that "Example Domain" is present

DONE.' > smoke.hunt

manul smoke.hunt
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `manul: command not found` | Run `make install` and ensure `~/.local/bin` is on `PATH`, or use `./manul`. |
| Chrome/Chromium not found | Install Google Chrome or Chromium and ensure it is on `PATH` (or set `--executable-path` / `MANUL_CHANNEL`). |
| Sandbox errors on Linux/CI | Add `--no-sandbox` via `MANUL_BROWSER_ARGS`. |
| Attach instead of launch | Start Chrome with `--remote-debugging-port=9222` and pass `--cdp http://127.0.0.1:9222`. |
