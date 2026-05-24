# AGENTS.md — hermes-web-computer

> Browser-native tiling AI desktop. Go backend + Svelte 5 frontend + WebSocket JSON-RPC.

## What It Is

**hermes-web-computer** is a tiling AI desktop served from a Linux server via web browser. Each "window" is a Svelte tile backed by a Go handler. The layout engine manages split/resize/focus. Tiles communicate through a JSON-RPC multiplexer over WebSocket.

**Vision:** "one website to rule them all" — a full desktop experience (Waybar, dock, workspaces, tiles) served from your server, accessible from any browser.

**App model:** Web-native tiles (Svelte+Go) as primary. xpra escape hatch for native Linux GUI apps that can't be rebuilt as web components.

**Multi-user:** Future — Coder integration for team workspace lifecycle (provision/suspend/delete per user).

---

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
  nohup ./agent-os server --port 3005 > /tmp/hwc-server.log 2>&1 &

# Run Go tests
go test ./... -count=1 -timeout=120s

# Screenshot for visual QA
google-chrome-stable --headless --disable-gpu --no-sandbox \
  --virtual-time-budget=30000 --window-size=1440,900 \
  --screenshot=/tmp/hwc-qa/screenshots/current.png \
  --disable-web-security http://localhost:3005
```

**Ports:** 3005 (HWC server), 3113 (legacy tunnel), 5174 (frontend dev)

---

## Project Structure

```
hermes-web-computer/
├── AGENTS.md              ← You are here
├── README.md              ← User-facing overview
├── docs/
│   ├── ARCHITECTURE.md    ← Master architecture (start here for deep work)
│   ├── SPEC.md             ← System specification (351 lines)
│   ├── ROADMAP.md          ← 6-phase development plan
│   ├── WAYBAR-SPEC.md      ← Waybar + workspaces + dock + file explorer spec
│   ├── ILLOGICAL-IMPULSE-DESIGN.md  ← Visual design system
│   ├── completion-plan.md  ← v1.0→v1.2 priorities
│   ├── ONE-WEBSITE.md      ← Vision doc
│   └── VISUAL-AUDIT-2026-05-23.md  ← Visual QA state
├── backend/                ← Go packages (see Backend Packages below)
├── frontend/
│   ├── src/
│   │   ├── App.svelte      ← Root layout (panels, dock, workspace pill)
│   │   ├── stores/
│   │   │   ├── ws.ts       ← WebSocket client + protocol helpers (critical)
│   │   │   ├── layout.svelte.ts   ← Layout tree state
│   │   │   └── workspace.svelte.ts ← 9-workspace state
│   │   └── components/     ← 31 Svelte 5 components
│   └── package.json
├── e2e/                    ← Playwright tests (17+ tests)
├── scripts/                ← Visual QA + build scripts
└── desktop/                ← Electron shell (Phase 5)
```

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25, `nhooyr.io/websocket`, `creack/pty` |
| Frontend | Svelte 5, Vite 6, Tailwind CSS 4 |
| Terminal | xterm.js (lazy-loaded) |
| Editor | Monaco Editor (lazy-loaded) |
| Browser automation | chromedp (headless Chrome) |
| Session storage | JSON files (`~/.hermes/hermes-web-computer/sessions/`) |
| Config | YAML (`~/.hermes/config.yaml`) |

---

## Backend Packages

All in `/home/sean/.hermes/hermes-web-computer/backend/`:

| Package | File | Purpose |
|---------|------|---------|
| **ws** | `multiplexer.go` | HTTP server + WebSocket endpoint + JSON-RPC routing (3 protocols: ui/agent/audio) |
| **ws** | `context.go` | ContextManager — tracks focused tile for agent context awareness |
| **ws** | `filesystem.go` | Sandboxed FS ops (list/read/write/stat/delete) — path restricted to HERMES_HWC_ROOT |
| **ws** | `apps.go` | App launcher — terminal (new PTY), editor, preview, browser (chromedp) |
| **agent** | `streamer.go` | SSE client for Hermes Agent — parses token/reasoning/tool_call/tool_result/stream_end |
| **pty** | `supervisor.go` | PTY lifecycle — creack/pty, 1MB ring buffer, checkpoint, signal handling |
| **layout** | `tree.go` | Binary tree layout engine — split/mount/unmount/resize/swap/fullscreen + delta ops |
| **session** | `store.go` | JSON file-based sessions — CRUD, message persistence, search, FTS index |
| **security** | `security.go` | Tiered YAML permissions (safe/prompt/block) + token-gated execution (30s expiry) |
| **browser** | `browser.go` | chromedp Manager — Navigate/Screenshot/Click/Input/Back/Forward/Eval |
| **llm** | `router.go` | Multi-provider LLM routing — OpenAI/Anthropic/Groq/Ollama/LMStudio |
| **mcp** | `client.go` | MCP JSON-RPC 2.0 stdio client — 10 operations, protocol version 2024-11-05 |
| **audio** | `bridge.go` | Fun-Audio-Chat binary protocol relay — Opus relay, text, interrupt |
| **config** | `manager.go` | config.yaml read/write + env vars |
| **docker** | `manager.go` | Docker CLI wrapper — list/stats/start/stop/restart/remove/logs |
| **state** | `state.go` | LayoutTree, Checkpoint, SessionState types |
| **telemetry** | `telemetry.go` | JSONL ring buffer + async HTTP sync with exponential backoff |

---

## Frontend Components

All in `/home/sean/.hermes/hermes-web-computer/frontend/src/components/`:

**Shell components:**
- `App.svelte` — Root layout (LeftPanel + MiddlePanel + RightPanel + Dock + WorkspacePill)
- `LeftPanel.svelte` — 280px resizable, tabs: Files/Apps/Sessions
- `RightPanel.svelte` — 360px resizable, tabs: Chat/Profiles/Skills/Crons/Memory/Settings/Observability
- `MiddlePanel.svelte` — Tile workspace (binary tree render)
- `Dock.svelte` — Floating bottom-center, 11 app icons, active dot indicator
- `WorkspacePill.svelte` — Top-center, 9 workspace dots + clock + agent status

**Tiles:**
- `Terminal.svelte` — xterm.js PTY terminal
- `Browser.svelte` — chromedp browser (navigate/screenshot/interact)
- `ChatPanel.svelte` — Hermes Agent chat with streaming
- `Monaco.svelte` — Monaco editor (lazy-loaded, read-only with Ctrl+S save)
- `Tile.svelte` — Tile container (lazy-mount on viewport entry)

**Dashboard tiles:**
- `DashOverview.svelte` — KPI cards, session analytics, event breakdown
- `DashFileManager.svelte` — Browse/preview/edit/create/delete files
- `DashSystemStatus.svelte` — System info, CPU/mem/disk, service status
- `DashAnalytics.svelte` — Token usage, daily breakdown, model/skill tables
- `DashObservability.svelte` — AI event feed, filters, status indicators

**Panels (RightPanel tabs):**
- `ProfilePanel.svelte` — Profile list with active indicator
- `SkillsPanel.svelte` — Skills grouped by category with filter
- `CronPanel.svelte` — Cron job list + create form + pause/resume/delete
- `MemoryPanel.svelte` — Memory read/write with two textareas
- `SettingsPanel.svelte` — 7-theme switcher
- `ObservabilityPanel.svelte` — Event feed with health indicators
- `ConfigPanel.svelte` — Model picker, env vars, restart signal

**Utilities:**
- `CommandPalette.svelte` — Fuzzy search, 8 categories, Ctrl+K
- `KeymapOverlay.svelte` — Keyboard shortcut overlay, Ctrl+?
- `FileTree.svelte` — File tree browser with path navigation
- `AppLauncher.svelte` — App launching UI with session management
- `SessionsPanel.svelte` — Session search and management
- `ResizeHandle.svelte` — Drag-to-resize handle
- `MiddlePanel.svelte` — Middle panel with drop target state

---

## WebSocket Protocol

**Endpoint:** `ws://localhost:3005/ws`

**JSON-RPC Envelope:**
```json
{"protocol": "ui|agent|audio", "method": "method.name", "params": {...}, "id": "req_1", "ts": 1234567890}
```

**Server Events:**
```json
{"protocol": "ui|agent|audio", "event": "event.name", "data": {...}, "ts": 1234567891}
```

### Protocol: ui (Layout & System)
| Method | Description |
|--------|-------------|
| `layout.update` | Apply layout op (split/mount/unmount/resize/swap/fullscreen) |
| `interrupt` | Shift+Space: checkpoint + SIGINT + amber border |
| `approval.grant` | Approve token-gated command |
| `fs.list`, `fs.read`, `fs.write`, `fs.stat`, `fs.delete` | Filesystem operations |
| `apps.list`, `apps.launch` | App listing and launch |
| `dashboard.stats`, `analytics.get` | Dashboard/analytics |
| `system.info`, `system.resources`, `system.services` | System info |
| `system.restart` | Restart signal |
| `session.new`, `session.list`, `session.get`, `session.delete`, `session.update` | Session management |
| `docker.list`, `docker.stats`, `docker.start`, `docker.stop`, `docker.restart`, `docker.remove`, `docker.logs` | Docker management |
| `config.get`, `config.set`, `config.delete` | Config management |
| `env.list`, `env.set`, `env.delete` | Environment variables |

### Protocol: agent (PTY, Chat, Browser, Tools)
| Method | Description |
|--------|-------------|
| `pty.write` | Write to PTY (with security classification: safe/prompt/block) |
| `chat.send` | Send message to Hermes Agent SSE |
| `tool.execute` | Execute tool via Hermes `/v1/chat/completions` |
| `browser.navigate`, `screenshot`, `click`, `input`, `back`, `forward` | Browser automation |
| `mcp.*` | All MCP methods (list/connect/disconnect/tools.list/tools.call/resources.list/resources.read/prompts.list/prompts.get) |

### Protocol: audio (Fun-Audio-Chat Relay)
| Method | Description |
|--------|-------------|
| `audio.start`, `audio.stop` | Session management |
| `audio.stream` | Opus chunk relay |
| `audio.interrupt` | Interrupt audio generation |
| `audio.text` | Send text to audio agent |

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Shift+Space` | Universal interrupt (pause/resume agent) |
| `Ctrl+K` | Command palette |
| `Ctrl+?` | Keymap overlay |
| `Ctrl+B` | Toggle left panel |
| `Ctrl+Shift+B` | Toggle right panel |
| `Shift+Arrow` | Move focus between tiles |
| `Shift+Alt+Arrow` | Resize tile borders |
| `Shift+D` | Cycle layout modes (master-stack, even-split, columns, rows) |
| `Shift+F` | Toggle fullscreen on focused tile |
| `Shift+Q` | Close focused tile |
| `Shift+Space` | Toggle floating/tiled mode on focused tile |
| `Shift+1-9` | Switch workspace |
| `Shift+Alt+1-9` | Move focused tile to another workspace |

---

## Current State (as of 2026-05-24)

### ✅ Complete (v1.2)
- WebSocket multiplexer (ui/agent/audio protocols)
- PTY supervisor with 1MB ring buffer + checkpoint
- Layout tree (split/mount/unmount/resize/swap/fullscreen with delta ops)
- Security enforcer (YAML config, safe/prompt/block tiers, token gating)
- Session store (JSON file-based, CRUD + message persistence)
- Browser tile (chromedp headless Chrome)
- 9 workspaces with independent layout trees
- Full keyboard shortcut map
- Theme: solid #191919 dark (all regions ΔE<6 vs reference)
- 45+ Go backend tests passing
- 17+ Playwright E2E tests passing

### ❌ Not Started
- **Waybar** — workspace indicators + window title + system tray (wifi/volume/battery/temp/clock)
- **Clickable workspace indicators** — currently keyboard-only (Shift+1-9)
- **Dock pinned apps + running indicators** — dock exists but apps launch panel, not tiles
- **File explorer sidebar** — needs VSCode-style collapsible tree
- **Menu bar** — File/Edit/View/Go/Run/Terminal/Help
- **Bottom terminal panel with tabs** — Terminal/Problems/Output/Ports
- **Workspace→app bindings** — $brow=firefox style config
- **Multi-user support** — Coder integration for team workspaces

### 🚧 In Progress
- Phase engine for remaining features (persistent state machine)

---

## Key Files to Read Before Work

| File | Why |
|------|-----|
| `docs/SPEC.md` | Master system specification |
| `docs/ARCHITECTURE.md` | Full architecture breakdown |
| `docs/WAYBAR-SPEC.md` | Waybar + workspaces + dock spec |
| `backend/ws/multiplexer.go` | Protocol routing — read first for any backend work |
| `frontend/src/stores/ws.ts` | WebSocket client — read first for any frontend work |
| `frontend/src/App.svelte` | Root layout — understand panel structure |
| `frontend/src/components/Dock.svelte` | Dock app launcher pattern |
| `frontend/src/components/WorkspacePill.svelte` | Workspace indicator pattern |

---

## Multi-User Architecture (Future)

```
┌─────────────────────────────────────────────────────────────────┐
│                     HWC Multi-User Architecture                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Browser #1 (User A)  │  Browser #2 (User B)  │  Browser #N    │
│  ┌──────────────────┐ │  ┌──────────────────┐ │  ┌───────────┐ │
│  │ HWC Shell         │ │  │ HWC Shell         │ │  │ HWC Shell │ │
│  │ (own workspace)   │ │  │ (own workspace)   │ │  │ (own ws)  │ │
│  └────────┬─────────┘ │  └────────┬─────────┘ │  └─────┬─────┘ │
│           │ WebSocket  │           │ WebSocket  │        │       │
│           └────────────┼───────────┼────────────┴────────┘       │
│                        │            │                            │
│              ┌─────────▼────────────▼──────────┐               │
│              │      Go Backend (HWC Multiplexer) │               │
│              │  ┌─────────────────────────────┐ │               │
│              │  │ Per-user session isolation   │ │               │
│              │  │ Workspace binding           │ │               │
│              │  │ Layout tree per connection  │ │               │
│              │  └─────────────────────────────┘ │               │
│              └──────────────┬───────────────────┘               │
│                             │                                    │
│              ┌──────────────▼──────────────┐                     │
│              │   Coder (Workspace Lifecycle) │                   │
│              │  Provision / Suspend / Delete │                   │
│              │  Per-user Linux VM            │                   │
│              └───────────────────────────────┘                   │
└──────────────────────────────────────────────────────────────────┘
```

**Coder integration:**
- OIDC auth via Keycloak
- Workspace = per-user Linux VM (persistent or on-demand)
- HWC workspace switches map to Coder workspace
- File sync between HWC and Coder workspaces via shared mount

---

## Native App Escape Hatch (xpra)

For apps that genuinely need native Linux GUI and can't be rebuilt as web tiles:

```
Native Linux App → X11 → xpra server → HTML5 → browser iframe
                    ↑
        HWC tile chrome around iframe
        Window position managed by HWC layout tree
```

xpra sessions embedded as HWC iframe tiles. HWC provides window management around xpra iframe. Use SSH tunneling from browser to server's xpra port.

**NOT Webtop** — Webtop runs a full Linux desktop in a container (XFCE/KDE/Sway) + noVNC. This is a separate product concern (persistent desktop sessions), not the tiling compositor model HWC uses.

---

## Visual Design

**Theme:** Solid `#191919` dark (confirmed ΔE<8 vs reference, 2026-05-24)
**Glassmorphism:** Removed — was causing purple tint and brightness mismatch

| Region | Color |
|--------|-------|
| Top bar | #191919 |
| Left panel | #191919 |
| Center tiles | #191919 |
| Right panel | #1d1d1d |
| Dock | #191919 |

**Font:** JetBrains Mono (terminal), system-ui (UI)
**Border radius:** 12px (tiles), 16px (floating panels)
**Shadows:** diffused panel shadows, no harsh drop shadows

---

## Testing

```bash
# Go backend tests
cd /home/sean/.hermes/hermes-web-computer/backend
go test ./... -count=1 -timeout=120s

# Playwright E2E
cd /home/sean/.hermes/hermes-web-computer/frontend
npx playwright test e2e/tests/01-layout.spec.ts

# Visual QA (on host)
google-chrome-stable --headless --disable-gpu --no-sandbox \
  --virtual-time-budget=30000 --window-size=1440,900 \
  --screenshot=/tmp/hwc-qa/screenshots/current.png \
  --disable-web-security http://localhost:3005
```

---

## Related Documentation

| Doc | Location | Purpose |
|-----|----------|---------|
| System spec | `docs/SPEC.md` | Master spec (351 lines) |
| Architecture | `docs/ARCHITECTURE.md` | Full architecture breakdown |
| Waybar spec | `docs/WAYBAR-SPEC.md` | Waybar + workspaces + dock + file explorer |
| Visual design | `docs/ILLOGICAL-IMPULSE-DESIGN.md` | Design tokens, glassmorphism, CSS |
| 6-phase roadmap | `docs/ROADMAP.md` | Development plan |
| Visual audit | `docs/VISUAL-AUDIT-2026-05-23.md` | QA state, design token audit |
| This repo's agent skill | `~/.hermes/skills/hermes-computer/SKILL.md` | Full operations guide |

---

## Notes

- **Path sandboxing:** `sanitizePath()` strips leading `/` and rejects `../` traversal. Even `/etc/shadow` becomes `allowedRoot/etc/shadow`.
- **Import block corruption:** When patching `multiplexer.go` with multiple agents, the `import (...)` block can get corrupted. Always `go build` to verify.
- **Subagent timeouts:** Subagents complete code changes but timeout before committing. Always `git status --short` after timeout — work is on disk.
- **vision_analyze:** Can only access local files from subagent runtime with `toolsets: ["vision"]`, not from execute_code context.
- **Workspace isolation:** Each workspace has an independent layout tree. Workspace switching sends `layout.update` with the workspace's saved tree.