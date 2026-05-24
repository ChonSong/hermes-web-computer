# Architecture — hermes-web-computer

> **Agent-OS v1.2** — A browser-based tiling AI desktop for collaborative development between a human, Hermes (text/terminal agent), and Fun-Audio-Chat (voice agent).

## Overview

hermes-web-computer is a **Go + Svelte 5** tiling desktop that runs entirely in the browser. The backend owns canonical state; the frontend renders deltas. All communication—UI events, agent tool calls, and audio streams—multiplexes over a single WebSocket connection using a JSON-RPC envelope protocol.

```
┌──────────────────────┐     WebSocket JSON-RPC (JSON envelope + protocol tag)     ┌───────────────────┐
│   Svelte 5 SPA       │ ◄──────────────────────────────────────────────────────► │   Go Backend       │
│   (Capture Phase)    │     {"protocol":"ui|agent|audio", "method":..., ...}      │   (Single Loop)    │
└──────────┬───────────┘                                                         └─────────┬─────────┘
           │                                                                           │
  ┌────────▼────────┐                                                     ┌──────────▼──────────┐
  │ Layout Renderer │                                                     │  PTY Supervisor     │
  │ Recursive Tiles │                                                     │  Ring Buffer        │
  │ +xterm.js       │                                                     │  Cgroups + PID NS   │
  └────────────────┘                                                     └──────────┬──────────┘
                                                                                  │
                                                                         ┌────────▼────────┐
                                                                         │ Hermes Agent   │
                                                                         │ Docker/Subproc│
                                                                         │ Fun-Audio-Chat │
                                                                         └────────────────┘
```

---

## Five Key Principles

| # | Principle | Detail |
|---|-----------|--------|
| 1 | **Backend owns truth** | Client renders deltas; zero layout drift |
| 2 | **Interrupt < 100ms** | `Shift+Space` → optimistic UI freeze + atomic checkpoint |
| 3 | **One wire** | JSON-RPC multiplexes UI, agent tools, and audio over a single WebSocket |
| 4 | **Lean by default** | No Temporal, no CRDTs, no AST parsers; opt-in only |
| 5 | **Voice-native** | Fun-Audio-Chat direct Opus stream, full-duplex interrupt |

---

## Project Structure

```
hermes-web-computer/
├── backend/                     # Go 1.26+ backend (single binary)
│   ├── cmd/server/main.go       # Entry point; wires all packages
│   ├── ws/                      # WebSocket multiplexer + JSON-RPC routing
│   │   ├── multiplexer.go       # Core: Envelope/Event types, protocol routing, session map
│   │   ├── context.go           # Focused tile context for agent awareness
│   │   ├── apps.go              # App launch handlers (terminal, editor, browser)
│   │   └── filesystem.go        # File operations over WebSocket
│   ├── pty/                     # PTY supervisor — ring buffer, checkpoint, signals
│   │   └── supervisor.go        # PTYSession, RingBuffer, SIGSTOP/SIGCONT
│   ├── layout/                  # Binary-tree layout engine
│   │   └── tree.go              # split/mount/unmount/resize/swap/fullscreen ops + delta generation
│   ├── state/                   # Session state — layout tree + checkpoints
│   │   └── state.go             # SessionState, Checkpoint, backend-owned truth
│   ├── security/                # YAML permissions + token-gated execution
│   │   └── security.go          # Enforcer, LoadConfig, UseDefaults
│   ├── browser/                 # Chromedp browser manager
│   │   └── browser.go           # Navigate, click, screenshot, DOM query
│   ├── audio/                   # Fun-Audio-Chat WebSocket relay (Opus binary protocol)
│   │   └── bridge.go             # Bridge, AudioSession, Connect, RelayAudio
│   ├── agent/                   # Hermes Agent SSE streamer
│   │   └── streamer.go          # Streamer, StreamEvent, ToolCall
│   ├── llm/                     # LLM router (single Hermes Agent API)
│   │   └── router.go
│   ├── mcp/                     # MCP client manager
│   │   └── client.go
│   ├── session/                 # Persistent session store
│   │   └── store.go             # Store, session persistence to disk
│   ├── config/                  # Configuration manager
│   │   └── manager.go
│   ├── docker/                  # Docker container manager
│   │   └── manager.go
│   ├── telemetry/               # JSONL ring buffer + async cloud sync
│   │   └── telemetry.go         # RingBuffer, Syncer
│   └── go.mod                   # Go 1.26, nhooyr.io/websocket, chromedp, creack/pty
├── frontend/                    # Svelte 5 + Vite + Tailwind CSS
│   ├── src/
│   │   ├── App.svelte           # Root: LeftPanel + MiddlePanel + RightPanel + Dock
│   │   ├── main.ts              # Entry point
│   │   ├── app.css              # Tailwind base + global resets
│   │   ├── stores/              # 9 Svelte 5 rune-less writable stores
│   │   │   ├── ws.ts            # WebSocket client + Envelope/Event helpers + layout ops
│   │   │   ├── layout.svelte.ts # Layout reactive state (derived from ws store)
│   │   │   ├── workspace.ts     # 9-workspace state, localStorage persistence
│   │   │   ├── sessions.svelte.ts
│   │   │   ├── config.svelte.ts
│   │   │   ├── crons.svelte.ts
│   │   │   ├── memory.svelte.ts
│   │   │   ├── profiles.svelte.ts
│   │   │   └── skills.svelte.ts
│   │   └── components/          # 28 Svelte 5 components
│   │       ├── App.svelte       # Root shell
│   │       ├── Tile.svelte      # Leaf renderer (xterm / monaco / browser / welcome)
│   │       ├── Dock.svelte      # App launcher dock (bottom bar)
│   │       ├── Terminal.svelte  # xterm.js PTY client
│   │       ├── Monaco.svelte    # Monaco editor client
│   │       ├── Browser.svelte   # Chromium screenshot client
│   │       ├── LeftPanel.svelte # File tree + sessions + skills + memory + crons
│   │       ├── MiddlePanel.svelte # Tile area + command palette
│   │       ├── RightPanel.svelte # Chat + observability + config panels
│   │       ├── ChatPanel.svelte  # Agent chat interface
│   │       ├── CommandPalette.svelte
│   │       ├── KeymapOverlay.svelte
│   │       ├── WorkspacePill.svelte # Workspace switcher pills
│   │       ├── ResizeHandle.svelte
│   │       ├── SessionsPanel.svelte
│   │       ├── SkillsPanel.svelte
│   │       ├── MemoryPanel.svelte
│   │       ├── ConfigPanel.svelte
│   │       ├── CronPanel.svelte
│   │       ├── ObservabilityPanel.svelte
│   │       ├── ProfilePanel.svelte
│   │       ├── AppLauncher.svelte
│   │       ├── FileTree.svelte
│   │       └── 9× Dash*.svelte  # Dashboard bento cards (Overview, FileManager, Analytics, ...)
│   ├── package.json             # Svelte 5, Vite, xterm.js, monaco-editor
│   └── vite.config.ts
├── bridge/                      # Python — Fun-Audio-Chat subprocess wrapper
├── bench/                       # Go — interrupt latency benchmark harness
├── deploy/                      # Docker Compose + Caddyfile
├── bench/                       # Go interrupt latency benchmark harness
└── docs/                        # This file + spec + decision logs
```

---

## Backend

### WebSocket Multiplexer (`ws/multiplexer.go`)

The `Multiplexer` is the core of the backend. A single `*websocket.Conn` per session is held open; all messages are JSON-RPC envelopes tagged by `protocol`:

```go
type Envelope struct {
    Protocol string          `json:"protocol"` // "ui", "agent", "audio"
    Method   string          `json:"method"`
    Params   json.RawMessage `json:"params,omitempty"`
    ID       string          `json:"id"`
    Ts       int64           `json:"ts"`
}

type Event struct {
    Protocol string          `json:"protocol"`
    Event    string          `json:"event"`
    Data     json.RawMessage `json:"data,omitempty"`
    Ts       int64           `json:"ts"`
}
```

The multiplexer owns the `Session` map (`sync.RWMutex`), a `*pty.Supervisor`, `*state.SessionState`, `*layout.LayoutTree`, `*security.Enforcer`, `*telemetry.RingBuffer`, `*audio.Bridge`, `*browser.Manager`, and a `*session.Store`. All routing is synchronous within the server's single event loop.

### PTY Supervisor (`pty/supervisor.go`)

- Manages multiple `PTYSession` entries keyed by PTY ID.
- Each `PTYSession` wraps an `exec.Cmd` with a `*os.File` pty, a 1 MB `RingBuffer` for checkpoint/replay, and a `chan []byte` output stream.
- `RingBuffer` is a circular byte buffer; on overflow the oldest bytes are dropped (tail-truncating, not head-dropping).
- Provides `Checkpoint()` → returns ring buffer contents + current cursor offset for atomic interrupt.
- Supports `SIGSTOP`/`SIGCONT` for pause/resume with cgroup + pid namespace isolation.

### Layout Engine (`layout/tree.go`)

Binary-tree layout engine. Nodes are either `leaf` (single content tile) or `split` (horizontal or vertical container with two children). Supported operations:

| Op | Description |
|----|-------------|
| `split` | Convert leaf → split with original + new child |
| `mount` | Attach a new leaf (used for app launch) |
| `unmount` | Detach a leaf |
| `resize` | Adjust relative `Size` (0.0–1.0) of a child |
| `swap` | Swap two leaf positions |
| `fullscreen` | Expand a leaf to fill the tree |

`Apply(op)` returns a list of canonical delta ops for the client. The tree is SHA-256 hashed for sync verification.

### State (`state/state.go`)

Backend-owned canonical session state:

```go
type SessionState struct {
    LayoutVersion int64
    LayoutHash    string
    Tree          LayoutTree   // mirrors layout tree
    ActiveTile    string
    AgentState    string       // "idle", "running", "paused", "error"
    ResumePolicy  string       // "A", "B", or "C"
}
```

```go
type Checkpoint struct {
    CursorOffset     int
    BufferHashes     map[string]string
    AgentPromptStack []string
    CWD              string
    Policy           string
    Ts               int64
}
```

Checkpoints are atomic JSON snapshots taken at `Shift+Space` interrupt time. Resume policy `A` = restart from checkpoint, `B` = replay ring buffer then continue, `C` = checkpoint + restart LLM.

### Security (`security/security.go`)

YAML-backed permission enforcer. Loads `~/.agent-os/security.yaml` on startup, falls back to permissive defaults if absent. Supports token-gated execution for privileged agent tool calls.

### Browser (`browser/browser.go`)

Chromedp-based browser manager. Methods: `Navigate(url)`, `Click(sel)`, `Screenshot()`, `Eval(expr)`. Used for agent web-browsing tool calls and automated testing.

### Audio Bridge (`audio/bridge.go`)

WebSocket relay between the multiplexer and Fun-Audio-Chat at `ws://localhost:11235/api/chat`. Handles Opus binary frames. Full-duplex: client sends audio chunks, server streams back transcriptions + synthesized voice.

### Agent Streamer (`agent/streamer.go`)

Calls the Hermes Agent SSE endpoint (`http://localhost:8642` by default). Yields `StreamEvent` tokens including `ToolCall` objects. Session cookie is forwarded for auth.

### LLM Router (`llm/router.go`)

Routes LLM requests to the configured provider. Currently single-Hermes (no fan-out).

### MCP Manager (`mcp/client.go`)

Manages MCP (Model Context Protocol) client connections for extended tool schemas.

### Session Store (`session/store.go`)

Persists session state to disk at `~/.agent-os/` (configurable via `AGENT_OS_STATE_DIR`). Session snapshots survive server restarts.

### Docker Manager (`docker/manager.go`)

Manages Docker containers for isolated agent execution environments.

### Telemetry (`telemetry/telemetry.go`)

JSONL ring buffer (`/agent/.telemetry/events.jsonl`, 100-entry) + async cloud syncer with exponential backoff.

### Config Manager (`config/manager.go`)

Handles runtime configuration loading and hot-reload.

---

## Frontend

### Store Architecture

All stores are Svelte 5-compatible writable stores (not `$state` runes — rune-less stores for broad compatibility). The primary store is `stores/ws.ts` which owns the WebSocket connection:

```
ws.ts store
├── connected: boolean
├── lastError: string | null
└── LayoutTree + layout ops (split, mount, resize, swap)

Derived reactive stores (synced from ws.ts):
├── layout.svelte.ts   → reactive layout tree for components
├── workspace.ts      → 9 workspaces, localStorage persistence
├── sessions.svelte.ts
├── config.svelte.ts
├── crons.svelte.ts
├── memory.svelte.ts
├── profiles.svelte.ts
└── skills.svelte.ts
```

### Svelte 5 Components (28 total)

**Shell & Layout:**
- `App.svelte` — Root shell: three-panel layout (LeftPanel / MiddlePanel / RightPanel), keyboard nav, workspace pills, command palette, keymap overlay, dock
- `Tile.svelte` — Renders a leaf tile: xterm, monaco, browser, or welcome content
- `MiddlePanel.svelte` — Tile area container, layout flash, tile focus navigation
- `LeftPanel.svelte` — Left sidebar (280px default, resizable): file tree, sessions, skills, memory, crons
- `RightPanel.svelte` — Right panel (360px default, resizable): chat, observability, config tabs
- `ResizeHandle.svelte` — Drag-to-resize handles between panels

**Dock & Navigation:**
- `Dock.svelte` — Bottom app launcher bar with pinned apps, drag-to-reorder
- `CommandPalette.svelte` — `Ctrl+K` fuzzy command search
- `KeymapOverlay.svelte` — `?` keyboard shortcut overlay
- `WorkspacePill.svelte` — Clickable workspace switcher pills (1–9)
- `AppLauncher.svelte` — App grid for dock

**Tile Content:**
- `Terminal.svelte` — xterm.js PTY client; connects PTY output stream to terminal display
- `Monaco.svelte` — Monaco Editor client with file path binding
- `Browser.svelte` — Chromium screenshot viewer with address bar
- `ChatPanel.svelte` — Agent chat interface ( Hermes text agent)
- `SessionsPanel.svelte` — Session list and restore
- `SkillsPanel.svelte` — Configurable agent skills
- `MemoryPanel.svelte` — Persistent memory / context management
- `CronPanel.svelte` — Scheduled task management
- `ObservabilityPanel.svelte` — System metrics, logs, trace viewer
- `ConfigPanel.svelte` — Runtime configuration editor
- `ProfilePanel.svelte` — User profile settings

**Dashboard (Bento Cards):**
- `DashOverview.svelte` — System overview bento grid
- `DashFileManager.svelte` — File manager bento card
- `DashAnalytics.svelte` — Analytics bento card
- `DashObservability.svelte` — Observability bento card
- `DashSystemStatus.svelte` — System status bento card

**Supporting:**
- `FileTree.svelte` — Hierarchical file browser

### UI / Visual Design

- **Theme:** Solid `#191919` dark backgrounds, `bg-[#191919]` used throughout all panels, cards, dock, and tiles
- **Styling:** Tailwind CSS v4 (imported in `app.css` via `@import "tailwindcss"`)
- **Transparency:** `backdrop-blur-xl` on floating panels (`bg-[#191919]/` implied)
- **Borders:** `border border-white/10` hairline separators
- **Shadows:** `shadow-panel` custom class for elevated panels

### Workspace System

9 workspaces (`workspace.ts`), indexed 1–9, each with its own `LayoutTree` and floating tile map. Active workspace persists to `localStorage` key `hwc-workspaces-v1`. Workspace switcher pills in the top bar enable instant switching.

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Shift+Space` | Interrupt agent (atomic checkpoint) |
| `Ctrl+K` | Command palette |
| `?` | Keymap overlay |
| `←→↑↓` | Tile focus navigation |
| `1–9` | Switch workspace |
| `Shift+D` | Cycle layout mode (master-stack / even-split / columns / rows) |

---

## WebSocket Protocol

All messages flow through a single WebSocket endpoint at `/ws`. The JSON-RPC envelope schema:

```
Client → Server (Envelope):
{
  "protocol": "ui" | "agent" | "audio",
  "method": "layout.split" | "pty.write" | "agent.complete" | "audio.send" | ...,
  "params": { ... },
  "id": "req-uuid",
  "ts": 171...000
}

Server → Client (Event — push):
{
  "protocol": "ui" | "agent" | "audio",
  "event": "layout.change" | "pty.output" | "agent.token" | "audio.frame" | ...,
  "data": { ... },
  "ts": 171...000
}

Server → Client (Envelope — response):
{
  "protocol": "ui" | "agent" | "audio",
  "method": "...",       // echoes the request method
  "result": { ... },
  "id": "req-uuid",
  "ts": 171...000
}
```

---

## Concurrency Model

The Go backend uses a **single-event-loop-per-connection** model:

1. `wsServe` goroutine: accepts WebSocket connections, registers `Session` in map, starts `wsRW` goroutine.
2. `wsRW` goroutine: reads `Envelope`, dispatches to `handleProtocol()`.
3. `handleProtocol()`: switches on `Envelope.Protocol`, routes to the appropriate handler (no locks needed within the same goroutine).
4. PTY output, audio frames, and agent tokens arrive asynchronously via channels and are **sent** to the client from within the same connection's `send` goroutine (via `Session.send <- Event`), keeping delivery order deterministic.

No mutex is held across protocol handler calls. Shared state (`sessions map`, `supervisor.ptys`, etc.) is protected by `sync.RWMutex` only for cross-goroutine access (e.g., a PTY output goroutine writing to the send channel while the main goroutine reads the map).

---

## Testing

- **Backend:** 45+ Go tests covering ws, pty, layout, security, session, audio bridge, state, context manager, filesystem, and app launch handlers.
- **e2e:** Playwright test suite (`e2e/`, `playwright.config.ts`).
- **Interrupt benchmark:** `bench/interrupt_test.go` measures `Shift+Space` → agent pause latency.

---

## Future Roadmap

| Feature | Description |
|---------|-------------|
| **Multi-user via Coder** | Hosted workspace provisioning for teams |
| **Waybar integration** | Linux status bar widget for workspace indicators |
| **Clickable workspace pills** | Direct workspace launch from the pill bar |
| **Dock pinned apps** | Persist pinned apps to localStorage |
| **File explorer sidebar** | Standalone file browser panel |
| **Bottom terminal tabs** | Tabbed terminal sessions in a single tile |

---

## Quick Start

```bash
# Full stack with make
make dev

# Manual (two terminals)
cd backend && go mod tidy && go run cmd/server/main.go   # :3001
cd frontend && npm install && npm run dev                # :5173

# Open http://localhost:3001 (prod) or http://localhost:5173 (dev)
```