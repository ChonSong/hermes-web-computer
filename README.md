# hermes-web-computer

> **hermes-web-computer v1.2** — A browser-based tiling AI desktop for collaborative development between a human, Hermes (text/terminal agent), and Fun-Audio-Chat (voice agent). Web-native tiles (Svelte+Go) as primary model; xpra escape hatch for native Linux GUI apps.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte)](https://svelte.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/ChonSong/hermes-web-computer/actions/workflows/ci.yml/badge.svg)](https://github.com/ChonSong/hermes-web-computer/actions)

---

## Philosophy

**Lean but Powerful.** No Temporal, no CRDTs, no AST parsers, no heavy telemetry sync. Backend-owned truth, sub-100ms interrupt, zero protocol bloat.

## Quick Start

```bash
# SSH to host (EndeavourOS)
ssh -i /home/hermeswebui/.hermes/container_key sean@172.19.0.1

# Build backend
cd /home/sean/.hermes/hermes-web-computer/backend
go build -o /tmp/hwc-server ./cmd/server/

# Build frontend
cd /home/sean/.hermes/hermes-web-computer/frontend && npm run build

# Start server (port 3005)
HERMES_HWC_ROOT=/home/sean/.hermes/hermes-web-computer \
  nohup ./hwc-server --port 3005 > /tmp/hwc-server.log 2>&1 &

# Run Go tests
cd /home/sean/.hermes/hermes-web-computer/backend
go test ./... -count=1 -timeout=120s
```

Open `http://localhost:3005` (port 3005, not 3001).

## Architecture

```
┌──────────────────────┐    WebSocket (JSON-RPC Multiplexer)     ┌───────────────────┐
│   Svelte 5 SPA       │ ◄─────────────────────────────────────► │   Go Backend      │
│   (Capture Phase)    │  {"protocol":"ui|agent|audio", ...}    │   (Single Loop)   │
└──────────┬───────────┘                                       └─────────┬─────────┘
           │                                                             │
   ┌───────▼────────┐                                           ┌───────▼──────────┐
   │ Layout Renderer│                                           │ PTY Supervisor   │
   │ Recursive Tiles│                                           │ Cgroups+PID NS   │
   └────────────────┘                                           └───────┬──────────┘
                                                                       │
                                                              ┌────────▼────────┐
                                                              │ Hermes / Audio  │
                                                              │ Docker+Subproc  │
                                                              └─────────────────┘
```

### Five Key Principles

| # | Principle | Detail |
|---|-----------|--------|
| 1 | **Backend owns truth** | Client renders deltas, zero layout drift |
| 2 | **Interrupt < 100ms** | `Shift+Space` → optimistic UI freeze + atomic checkpoint |
| 3 | **One wire** | JSON-RPC multiplexes UI, agent tools, and audio over a single WS |
| 4 | **Lean by default** | Cut Temporal, CRDTs, AST parsers; opt-in only |
| 5 | **Voice-native** | Fun-Audio-Chat direct Opus stream, full-duplex interrupt |

---

## Project Structure

```
├── backend/                     # Go backend
│   ├── cmd/server/main.go       # Entry point
│   ├── ws/                      # WebSocket multiplexer + JSON-RPC routing
│   │   ├── multiplexer.go       # Core: Envelope/Event types, protocol routing
│   │   └── apps.go              # App launch handlers (terminal, editor, browser)
│   ├── pty/                     # PTY supervisor — ring buffer, checkpoint, signals
│   ├── layout/                  # Layout tree — split/mount/unmount/resize/swap ops
│   ├── state/                   # Session state — layout tree + checkpoints
│   ├── security/                # YAML permissions + token-gated execution
│   ├── browser/                 # Chromedp browser manager — navigate, click, screenshot
│   ├── audio/                   # Fun-Audio-Chat WebSocket relay — Opus binary protocol
│   └── telemetry/               # JSONL ring buffer + async cloud sync with backoff
├── frontend/                    # Svelte 5 + Vite + xterm.js + Monaco
│   ├── src/
│   │   ├── stores/ws.ts         # WebSocket store + all protocol helpers
│   │   ├── components/          # Svelte components (19 files)
│   │   └── App.svelte           # Root layout: panels, dock, tiles
│   ├── package.json
│   └── vite.config.ts
├── deploy/                      # Docker Compose + Caddyfile
│   └── docker-compose.yml       # 4 services: agent-os, hermes, fun-audio, caddy
├── bridge/                      # Python — Fun-Audio-Chat subprocess wrapper
├── bench/                       # Go — interrupt latency benchmark harness
└── docs/                        # Spec + architecture diagrams + decision logs
```

---

## WebSocket Protocol

All communication flows through a single WebSocket endpoint at `/ws` using a JSON-RPC envelope with a `protocol` tag for multiplexing.

### Message Envelope (Client → Server)

```json
{
  "protocol": "ui" | "agent" | "audio",
  "method": "layout.update",
  "params": { "op": "split", "target_id": "root", "direction": "h" },
  "id": "req_1",
  "ts": 1710000000000
}
```

### Event (Server → Client)

```json
{
  "protocol": "ui",
  "event": "layout.delta",
  "data": { "layout_version": 2, "ops": [...] },
  "ts": 1710000000001
}
```

### Protocol: `ui` — Layout & System

| Method | Params | Description |
|--------|--------|-------------|
| `layout.update` | `{op, target_id, direction?, content?, pty_id?}` | Apply layout operation (split/mount/unmount/resize/swap/fullscreen) |
| `interrupt` | — | Trigger Shift+Space: checkpoint + SIGINT + amber border |
| `approval.grant` | `{token}` | Approve a security-gated command |
| `fs.list` | `{path}` | List directory contents |
| `fs.read` | `{path}` | Read file contents |
| `fs.write` | `{path, content, encoding}` | Write file |
| `fs.stat` | `{path}` | Get file metadata |
| `fs.delete` | `{path}` | Delete file or directory |
| `apps.list` | — | List launchable app types |
| `apps.launch` | `{type, path?}` | Launch app (terminal/editor/preview/browser) |
| `dashboard.stats` | — | Get session/uptime stats |
| `analytics.get` | `{days}` | Get analytics for N days (default 7) |
| `system.info` | — | Version, Go version, OS, arch |
| `system.resources` | — | Memory alloc, goroutines, CPU, GC pause |
| `system.services` | — | Status of websocket/pty/browser/audio |
| `observability.status` | — | Telemetry connected status |

**Server Events:** `layout.initial`, `layout.delta`, `border.state`, `agent.paused`, `approval.required`, `approval.granted`, `apps.list.response`, `apps.launch.response`, `apps.error`, `fs.*`, `dashboard.stats.response`, `analytics.result`, `system.*.response`, `observability.status`, `error`

### Protocol: `agent` — PTY, Chat, Browser

| Method | Params | Description |
|--------|--------|-------------|
| `pty.write` | `{data}` | Write to PTY (with 3-tier security: safe/prompt/block) |
| `chat.send` | `{message}` | Send message to Hermes Agent API (`HERMES_API_URL`) |
| `tool.execute` | — | Execute tool via Hermes (TODO) |
| `browser.navigate` | `{session_id, url}` | Navigate browser tile |
| `browser.screenshot` | `{session_id}` | Capture screenshot |
| `browser.click` | `{session_id, x, y}` | Click at coordinates |
| `browser.input` | `{session_id, text}` | Type text into browser |
| `browser.back` / `browser.forward` | `{session_id}` | Browser navigation |

**Server Events:** `pty.output`, `security.error`, `approval.required`, `command.blocked`, `chat.reply`, `chat.error`, `browser.navigated`, `browser.screenshot.response`, `browser.clicked`, `browser.input.done`, `browser.error`, `tool.execute`

### Protocol: `audio` — Fun-Audio-Chat Relay

| Method | Params | Description |
|--------|--------|-------------|
| `audio.start` | `{session_id?}` | Start audio session |
| `audio.stop` | — | Stop audio session |
| `audio.stream` | `{opus_chunk}` | Stream Opus audio chunk |
| `audio.interrupt` | — | Interrupt audio agent |
| `audio.text` | `{text}` | Send text to audio agent |

**Server Events:** `audio.started`, `audio.stopped`, `response`, `error`

### Security Model

Commands typed into the PTY are classified into 3 tiers:

| Tier | Behavior |
|------|----------|
| `safe` | Written directly to PTY |
| `prompt` | Generates approval token → UI shows approval dialog → `approval.grant` to execute |
| `block` | Blocked, red border shown |

Config: `~/.agent-os/security.yaml` (falls back to defaults if missing).

---

## Frontend Components (Svelte 5)

| Component | Description |
|-----------|-------------|
| `App.svelte` | Root layout — connects WS, renders panels/dock/overlay |
| `Tile.svelte` | Recursive layout tile — renders content based on node type (xterm/editor/browser/dash-*/welcome), max depth 3 |
| `Terminal.svelte` | xterm.js terminal attached to a PTY ID |
| `Monaco.svelte` | Monaco code editor for file editing |
| `Browser.svelte` | Browser tile — displays screenshots, handles navigation |
| `Dock.svelte` | macOS-style dock with 6 launchers: Files, Terminal, Agent, Browser, Dashboard, Voice |
| `CommandPalette.svelte` | Ctrl+K command palette — filtered list of layout operations |
| `KeymapOverlay.svelte` | Ctrl+? keyboard shortcuts reference overlay |
| `LeftPanel.svelte` | Left sidebar panel |
| `MiddlePanel.svelte` | Center panel |
| `RightPanel.svelte` | Right sidebar panel |
| `ResizeHandle.svelte` | Draggable resize handles between panels |
| `FileTree.svelte` | File tree explorer for DashFileManager |
| `AppLauncher.svelte` | App launcher UI |
| `WorkspacePill.svelte` | Workspace indicator pill |
| `DashOverview.svelte` | Dashboard: session stats, uptime |
| `DashFileManager.svelte` | Dashboard: file browser |
| `DashObservability.svelte` | Dashboard: telemetry status |
| `DashAnalytics.svelte` | Dashboard: usage analytics |
| `DashSystemStatus.svelte` | Dashboard: system resources/services |

### Stores (`stores/ws.ts`)

| Store/Function | Purpose |
|----------------|---------|
| `ws` (writable) | Connection state: `{connected, lastError}` |
| `layout` (writable) | Layout tree + version number |
| `focus` (writable) | Currently focused tile ID |
| `ptyOutputs` (writable) | Map of PTY ID → output string |
| `connect(url)` | Open WebSocket (auto-reconnect on close) |
| `send(envelope)` | Send a JSON-RPC envelope |
| `sendOp(op)` | Shorthand for `layout.update` |
| `on(event, handler)` | Subscribe to server events |
| `fsList/fsRead/fsWrite/fsStat/fsDelete` | Filesystem helpers |
| `appsList/appsLaunch` | App helpers |
| `chatSend` | Send message to Hermes agent |
| `audioStart/audioStop/audioStream` | Audio helpers |
| `dashStats/analyticsGet/systemInfo/systemResources/systemServices/observabilityStatus` | Dashboard helpers |

---

## Keyboard Shortcuts

| Shortcut | Action | Scope |
|----------|--------|-------|
| `Shift+Space` | Universal interrupt (checkpoint + SIGINT) | Global |
| `Shift+←→↑↓` | Move focus between tiles | Active tile |
| `Shift+D` | Swap split orientation (h↔v) | Active tile |
| `Shift+Alt+←→↑↓` | Resize tile | Active tile |
| `Shift+F` | Toggle fullscreen | Active tile |
| `Shift+Q` | Close tile | Active tile |
| `Ctrl+K` | Open command palette | Global |
| `Ctrl+?` | Open keyboard shortcuts overlay | Global |
| `Double-click` | Split tile horizontally | Any tile |
| `Escape` | Close overlays | Global |

---

## Design System

The UI uses a **solid dark** aesthetic (glassmorphism removed, 2026-05-23):

| Region | Color |
|--------|-------|
| Top bar / Left panel / Dock | `#191919` |
| Center tiles | `#1a1a1a` |
| Right panel | `#1d1d1d` |
| Border radius | 12px (tiles), 16px (floating panels) |

> Full design tokens: [`docs/ILLOGICAL-IMPULSE-DESIGN.md`](docs/ILLOGICAL-IMPULSE-DESIGN.md)

---

## Testing

### Unit Tests (Go)

```bash
cd backend && go test ./... -v
```

### Benchmarks (Interrupt Latency)

```bash
cd backend && go test ./bench -bench=. -benchmem
```

### E2E Test

The E2E test validates the full WebSocket lifecycle:

```
1. Connected
2. Layout received
3. Sending echo...
4. PTY: "echo HELLO_TEST"
5. Echo confirmed!
6. Interrupt: {"policy":"B","checkpoint_size":235}
=== ALL TESTS PASSED ===
```

---

## Deployment

### Docker Compose

```yaml
# deploy/docker-compose.yml — 4 services
#
# agent-os    → Go backend (:3001), state/telemetry volumes
# hermes      → Nous Hermes Agent (host network, GPU optional)
# fun-audio   → Fun-Audio-Chat (host network, NVIDIA GPU required)
# caddy       → Reverse proxy with auto-TLS, serves static frontend
```

```bash
cd deploy && docker compose up -d
```

### Build for Production

```bash
make build
# → backend/agent-os (static binary, CGO_ENABLED=0)
# → frontend/dist/ (static SPA)
```

The Go server serves the built `frontend/dist/` static files directly — no separate web server needed in production.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HERMES_API_URL` | `http://localhost:8642` | Hermes Agent API endpoint for chat |
| `FUN_AUDIO_WS` | `ws://localhost:11235/api/chat` | Fun-Audio-Chat WebSocket URL |
| `LOG_LEVEL` | `info` | Backend log level |
| `TELEMETRY_ENDPOINT` | (empty) | Cloud telemetry sync endpoint |

---

## Tech Stack

| Layer | Choice | Why |
|-------|--------|-----|
| Frontend | Svelte 5 + Vite + Tailwind | <50KB initial bundle, zero VDOM, runes |
| Backend | Go (`net/http` + `nhooyr.io/websocket`) | Sub-5MB RSS, native goroutines |
| PTY | `creack/pty` | Battle-tested, maintained |
| Storage | `modernc.org/sqlite` | Pure Go, no CGO |
| Browser | `chromedp` | Headless Chrome automation |
| Deploy | Docker Compose + Caddy | Auto-TLS, WebSocket proxy |
| Audio | Fun-Audio-Chat (WebSocket relay) | Native Opus, no MCP tax |

---

## Current Status (v1.2, 2026-05-24)

**Core:**
- ✅ WebSocket multiplexer (ui/agent/audio protocols, JSON-RPC)
- ✅ PTY supervisor — 1MB ring buffer + checkpoint + SIGINT
- ✅ Layout engine — split/mount/unmount/resize/swap/fullscreen + delta ops
- ✅ Security enforcer — YAML tiers (safe/prompt/block) + token gating
- ✅ Session store — JSON file-based, full-text search
- ✅ Browser tile — chromedp headless Chrome (navigate/click/input/screenshot)
- ✅ 9 workspaces with independent layout trees
- ✅ Full keyboard shortcut map

**UI:**
- ✅ Svelte 5 SPA (31 components: panels, tiles, dock, command palette)
- ✅ Theme: solid `#191919` dark (ΔE<8 vs reference, verified 2026-05-24)
- ✅ xterm.js terminal (lazy-loaded)
- ✅ Monaco editor (read-only with Ctrl+S save)
- ✅ 5 Dashboard tiles (Overview/FileManager/SystemStatus/Analytics/Observability)
- ✅ 17+ Playwright E2E tests (layout, resize, chaos, a11y, perf)
- ✅ 45+ Go backend tests

**What's Next:**
- 🚧 Waybar + clickable workspaces (see [`docs/WAYBAR-SPEC.md`](docs/WAYBAR-SPEC.md))
- 🚧 Dock refinements (click-to-launch tiles, running indicators)
- 🚧 File explorer sidebar (VSCode-style collapsible tree)
- 🚧 Bottom terminal panel (tabbed: Terminal/Problems/Output/Ports)
- 🚧 Multi-user support — Coder integration ([`docs/MULTI-USER-PLAN.md`](docs/MULTI-USER-PLAN.md))

## Full Specification

See [`docs/SPEC.md`](docs/SPEC.md) for the complete v1.2 specification and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full architecture breakdown.

## License

MIT
