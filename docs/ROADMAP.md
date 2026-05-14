# Agent-OS: Maximum Viable Product — Integrated Roadmap

> **Status:** Planning v1  
> **Goal:** All valuable features from hermes-webui + hermes-workspace + agent-os + hermes-web-computer, unified in one Go+Svelte5 codebase  
> **Principle:** Maximum viable, not minimum viable — but phased so something works at every stage

---

## Source of Truth for Features

This roadmap synthesizes the best features from four sources:

| Source | What's valuable |
|--------|----------------|
| `nesquena/hermes-webui` | Sessions (CRUD, FTS, projects, tags, CLI bridge), streaming SSE, slash commands, profiles, skills, cron, memory, themes (7), auth |
| `outsourc-e/hermes-workspace` | Electron desktop shell, TanStack Start routing, react-query, larger component library, swarm/multi-agent |
| `ChonSong/agent-os` | Docker/container observability, MCP server management, full observability dashboard, app registry, cron jobs UI, profiles UI |
| `ChonSong/hermes-web-computer` | Tiling WM, keyboard shortcuts, Go backend, Svelte5, WS multiplexer, PTY, audio bridge, shift+space interrupt |

**What to take from each:**
- hermes-webui → sessions, chat, streaming, profiles, skills, cron, memory, slash commands, themes, auth
- hermes-workspace → electron shell, routing, component patterns
- agent-os → docker management, observability, MCP management, deployment patterns
- hermes-web-computer → tiling WM, keyboard shortcuts, Go backend, WS multiplexer, PTY, interrupt

**What NOT to take:**
- hermes-workspace's 3D environment (Too specific, not broadly useful)
- agent-os's Postgres dependency (Go codebase should use SQLite or be file-based)
- hermes-webui's Python server (we're Go, not Python)
- hermes-webui's vanilla JS frontend (we're Svelte 5, not vanilla)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Browser (Svelte 5)                        │
│  ┌──────────┬─────────────────────────────┬──────────────────┐  │
│  │  LEFT    │        MIDDLE                │     RIGHT        │  │
│  │  280px   │        1fr (tiling)          │     360px        │  │
│  │          │                             │                  │  │
│  │ Files    │  ┌────────┬────────┐        │  Agent Chat      │  │
│  │ Apps     │  │Terminal│Monaco  │        │  Memory          │  │
│  │ Skills   │  ├────────┼────────┤        │  Voice Toggle    │  │
│  │ Sessions │  │Terminal│Terminal│        │                  │  │
│  │          │  └────────┴────────┘        │  Context Hist   │  │
│  └──────────┴─────────────────────────────┴──────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │ WS (single wire)
                    ┌─────────┴─────────┐
                    │  Go Backend       │
                    │  (3112)           │
                    │  ┌─────────────┐  │
                    │  │ Multiplexer │  │
                    │  │  ui/agent/  │  │
                    │  │  audio      │  │
                    │  └─────────────┘  │
                    │  ┌─────────────┐  │
                    │  │ PTY Supervisor│ │
                    │  │ (creack/pty) │  │
                    │  └─────────────┘  │
                    │  ┌─────────────┐  │
                    │  │ Hermes Agent │ │
                    │  │ (HTTP/8642) │ │
                    │  └─────────────┘  │
                    │  ┌─────────────┐  │
                    │  │ SQLite DB   │ │
                    │  │ (modernc)  │ │
                    │  └─────────────┘  │
                    └──────────────────┘
                              │
                    ┌─────────┴──────────┐
                    │  Docker Sandbox     │
                    │  (Playwright/CDP)   │
                    └────────────────────┘
```

---

## Phase 1: Foundation (2-3 weeks)

**Goal:** A working chat + sessions app with hermes-webui-quality UX, on the Go+Svelte5 stack.

### 1.1 Backend — Session Management (Go + SQLite)

**New files:**
- `backend/session/store.go` — SQLite-backed session store (read from `hermes-webui`'s `models.py` patterns, port to Go)
- `backend/session/handlers.go` — REST/WebSocket handlers for session CRUD

**Features:**
- Create, read, update, delete sessions
- Session persistence to SQLite (single `.db` file in `~/.hermes/agent-os/sessions.db`)
- Per-session workspace, model, created_at, updated_at, pinned, archived, project_id
- Full-text search via SQLite FTS5
- Session index file for fast listing
- Title auto-generation from first user message

**From hermes-webui:**
```go
// Session model — mirrors hermes-webui/models.py
type Session struct {
    ID        string    // 12-char hex (uuid4[:12])
    Title     string    // auto from first message, max 64 chars
    Workspace string    // absolute path, resolved at creation
    Model     string    // model ID (e.g. "anthropic/claude-sonnet-4")
    Messages  []Message // OpenAI-format message array
    CreatedAt int64     // Unix timestamp
    UpdatedAt int64     // Unix timestamp
    Pinned    bool
    Archived   bool
    ProjectID  string    // nullable
    ToolCalls  []ToolCall
}
```

### 1.2 Frontend — Session UI (Svelte)

**New files:**
- `frontend/src/components/SessionList.svelte` — left panel session list
- `frontend/src/components/SessionItem.svelte` — individual session row
- `frontend/src/components/ChatPanel.svelte` — message list + composer
- `frontend/src/components/MessageItem.svelte` — individual message
- `frontend/src/stores/sessions.ts` — session state management

**Features (from hermes-webui):**
- Sidebar session list with date grouping (Today/Yesterday/Earlier)
- Session search (full-text)
- Pin/unpin, archive, delete, duplicate sessions
- Session projects with colors (grouped in sidebar)
- Session tags (#tag in title → colored chips)
- Active session indicator in browser tab title

### 1.3 Streaming — SSE Chat to Hermes Agent

**File:** `backend/agent/streaming.go`

**From hermes-webui (`streaming.py`):**
- POST to Hermes `/v1/chat/completions` with SSE
- Stream tokens to frontend via WS multiplexer
- Tool call cards with expand/collapse
- Approval cards for dangerous commands (Tier 1)
- Cancel via stream_id tracking
- Turn journal audit trail

**WS protocol:**
```json
// Frontend → Backend
{"protocol": "agent", "method": "chat.send", "params": {"session_id": "...", "message": "...", "context": {}}}

// Backend → Frontend (streamed)
{"protocol": "agent", "event": "token", "data": {"text": "..."}}
{"protocol": "agent", "event": "tool", "data": {"name": "...", "preview": "..."}}
{"protocol": "agent", "event": "approval", "data": {"command": "...", "description": "..."}}
{"protocol": "agent", "event": "done", "data": {"session": {...}}}
{"protocol": "agent", "event": "error", "data": {"message": "..."}}
```

### 1.4 Profiles & Skills & Cron

**From hermes-webui:**
- Profile management (create, switch, delete, clone)
- Skill list/search/preview (read from disk `~/.hermes/skills/`)
- Cron job CRUD (create, pause, resume, trigger, delete)
- Memory: inline edit of `MEMORY.md` and `USER.md`

**New files:**
- `backend/profiles/handlers.go`
- `backend/skills/handlers.go`
- `backend/cron/handlers.go`
- `backend/memory/handlers.go`

**From agent-os (bonus):**
- Docker container list/status from Dockerode
- Cron job run history

### 1.5 Themes

**From hermes-webui:** 7 built-in themes (Dark, Light, Slate, Solarized Dark, Monokai, Nord, OLED)

**Implementation:** CSS custom properties per theme, persisted to `settings.json`

**Default:** Illogical Impulse glassmorphism (from `ILLOGICAL-IMPULSE-DESIGN.md`) as the "Dark" default

### 1.6 Auth

**From hermes-webui:** Optional HMAC cookie auth (24h TTL), login page at `/login`

**Implementation:** Simple optional auth — set password in env/config, get signed cookie

---

## Phase 2: Tiling Desktop (2-3 weeks)

**Goal:** hermes-web-computer's tiling WM merged into the session UI.

### 2.1 Middle Panel — Tiling Layout

**Replace** simple chat middle pane with **binary tree tiling:**

**From hermes-web-computer:**
- CSS Grid or flexbox binary tree
- Vertical/horizontal splits
- Maximum 2×2 grid per pane
- Resize via drag handles
- Swap split orientation

### 2.2 Tile Types

| Tile | Source | Status |
|------|--------|--------|
| Terminal (xterm.js) | hermes-web-computer | Existing |
| Monaco Editor | hermes-web-computer | Planned |
| Chat (message thread) | hermes-webui | Port from Phase 1 |
| Preview (file preview) | hermes-webui | Port from Phase 1 |
| Dashboard (observability) | agent-os | Port from Phase 1 |

### 2.3 Keyboard Shortcuts

**From SPEC.md (v2.0):**
- `Shift+Space`: Universal interrupt (pause/resume agent)
- `Shift+Arrow Keys`: Move focus between tiles
- `Shift+D`: Swap split orientation
- `Shift+Alt+Arrow Keys`: Resize borders
- `Shift+F`: Toggle fullscreen
- `Shift+Q`: Kill/close tile (auto-rebalance)
- `Ctrl+K`: Command palette
- `Ctrl+B` / `Ctrl+Shift+B`: Toggle left/right panels

### 2.4 Panel Resizing

- Left panel: 200px minimum, resizable 200-500px
- Right panel: 280px minimum, resizable 280-600px
- Persist widths to localStorage

---

## Phase 3: Observability & Management (1-2 weeks)

**Goal:** agent-os's Docker/container management + hermes-webui's system observability.

### 3.1 Docker Container Management

**From agent-os:**
- Container list (name, image, status, ports, created)
- Container logs (streaming via Dockerode)
- Container start/stop/restart
- Container stats (CPU, memory, network I/O)
- Inspect container (config, mounts, env)

**New files:**
- `backend/docker/manager.go` — Dockerode integration
- `backend/docker/handlers.go` — HTTP handlers for container ops

### 3.2 Observability Dashboard

**From agent-os:**
- `DashObservability.svelte` — event log, telemetry
- `DashSystemStatus.svelte` — container status, Hermes health
- JSONL ring buffer for agent actions

**New files:**
- `frontend/src/components/ObservabilityPanel.svelte`
- `backend/telemetry/ring.go` — JSONL ring buffer

### 3.3 Hermes Agent Config & Env

**From agent-os:**
- Config editor (YAML read/write via backend)
- Env var management (set, delete, reveal masked)
- Model picker (from Hermes `/v1/models`)

---

## Phase 4: Advanced Features (2-4 weeks)

### 4.1 Command Palette

- `Ctrl+K` opens global command palette
- Fuzzy search over: sessions, skills, commands, settings, files
- Keyboard navigation (arrow keys + enter)
- From hermes-webui slash commands (extended to global)

**New files:**
- `frontend/src/components/CommandPalette.svelte`
- `backend/commands/registry.go` — command discovery

### 4.2 MCP Adapter

**From SPEC.md:** Opt-in MCP adapter for interoperability

**Implementation:**
- MCP client in Go (connect to MCP servers via stdio)
- Expose MCP tools to Hermes agent
- MCP server management UI (from agent-os)

### 4.3 Voice Integration

**From SPEC.md:**
- MediaRecorder + Opus chunks via WS `audio` protocol
- Fun-Audio-Chat Python subprocess relay
- Full-duplex: detect human speech → pause agent

**Implementation:** `backend/audio/bridge.go` — relay audio to Python subprocess

### 4.4 LiteLLM Proxy

**From SPEC.md:**
- Swap Hermes provider without codebase changes
- 100+ model provider support via LiteLLM

**Implementation:** Configure as alternative to direct Hermes calls

### 4.5 Playwright Sandbox (Tiered Security)

**From SPEC.md:**
- Docker container with cgroup limits
- Headful browser automation via CDP
- Tier 0 (safe), Tier 1 (prompt), Tier 2 (block) security enforcement

**Implementation:**
- `backend/sandbox/docker.go` — container lifecycle
- `backend/security/enforcer.go` — regex allow-lists

---

## Phase 5: Desktop Shell (2-3 weeks)

**Goal:** Electron app wrapping the web UI for desktop integration.

**From hermes-workspace:**
- Electron with system tray
- Native menus
- OS notifications
- Auto-start on login

**New files:**
- `desktop/` — Electron project (separate from main repo or in `desktop/` subdir)

---

## Phase 6: Polish & Hardening (ongoing)

### 6.1 Testing (from PLAN.md Track 4)

| Suite | Tests | Frequency |
|-------|-------|-----------|
| Functional (layout, resize, sessions) | 10 | Every commit |
| Workflows (file edit, pipeline, chat) | 7 | Every commit |
| Chaos (server death, network, flood) | 4 | Nightly |
| Accessibility (keyboard, screen reader, contrast) | 3 | Every commit |
| Visual regression (pixel diff) | 2 | Every commit |
| Performance (load, memory, stability) | 3 | Nightly |

### 6.2 CI/CD

**From agent-os:**
- GitHub Actions: test → build → deploy
- Docker image build + publish to GHCR

### 6.3 hermes-webui Parity Checklist

| Feature | hermes-webui | Status | Notes |
|---------|-------------|--------|-------|
| Session CRUD | ✅ | Phase 1 | |
| Session search (FTS) | ✅ | Phase 1 | |
| Session projects/tags | ✅ | Phase 1 | |
| Streaming SSE | ✅ | Phase 1 | |
| Tool call cards | ✅ | Phase 1 | |
| Approval cards | ✅ | Phase 1 | |
| Profiles | ✅ | Phase 1 | |
| Skills | ✅ | Phase 1 | |
| Cron | ✅ | Phase 1 | |
| Memory | ✅ | Phase 1 | |
| Slash commands | ✅ | Phase 1 | |
| Themes (7) | ✅ | Phase 1 | |
| Auth | ✅ | Phase 1 | |
| Tiling WM | ❌ | Phase 2 | HWC-only |
| Keyboard shortcuts | ❌ | Phase 2 | HWC-only |
| Docker management | ❌ | Phase 3 | agent-os only |
| Observability | ❌ | Phase 3 | agent-os only |
| Command palette | ❌ | Phase 4 | |
| MCP | ❌ | Phase 4 | |
| Voice | ❌ | Phase 4 | |
| LiteLLM | ❌ | Phase 4 | |
| Playwright sandbox | ❌ | Phase 4 | |
| Electron shell | ❌ | Phase 5 | |

---

## File Manifest (New/Modified)

### Backend (Go)

```
backend/
├── cmd/server/main.go           # Entry point, port 3112
├── session/
│   ├── store.go                 # SQLite session store
│   ├── handlers.go              # Session CRUD handlers
│   └── index.go                # Session index file management
├── agent/
│   ├── streaming.go            # SSE → Hermes, token streaming
│   ├── protocol.go              # Agent WS protocol
│   └── client.go               # Hermes HTTP client
├── profiles/
│   └── handlers.go             # Profile CRUD
├── skills/
│   └── handlers.go             # Skill list/install/delete
├── cron/
│   ├── scheduler.go            # In-process cron (cronexpr)
│   └── handlers.go            # Cron CRUD
├── memory/
│   └── handlers.go             # MEMORY.md/USER.md read/write
├── docker/
│   ├── manager.go              # Dockerode integration
│   └── handlers.go             # Container ops
├── telemetry/
│   ├── ring.go                 # JSONL ring buffer
│   └── syncer.go               # Cloud sync (async)
├── security/
│   └── enforcer.go             # Regex allow-lists, Tier 0-2
├── sandbox/
│   └── docker.go               # Container lifecycle
├── audio/
│   └── bridge.go               # Fun-Audio-Chat relay
├── ws/
│   ├── multiplexer.go          # Existing — extend with sessions
│   ├── context.go             # Existing
│   └── filesystem.go          # Existing
└── go.mod
```

### Frontend (Svelte 5)

```
frontend/src/
├── components/
│   ├── SessionList.svelte       # Left panel: session list
│   ├── SessionItem.svelte      # Session row (pin/archive/delete)
│   ├── ChatPanel.svelte         # Message list + composer
│   ├── MessageItem.svelte       # Message with tool/approval cards
│   ├── CommandPalette.svelte    # Ctrl+K global palette
│   ├── ObservabilityPanel.svelte
│   ├── ResizeHandle.svelte     # Drag resize for columns
│   └── [existing components]
├── stores/
│   ├── sessions.ts             # Session state
│   ├── chat.ts                 # Message state
│   ├── profiles.ts
│   ├── skills.ts
│   ├── cron.ts
│   └── [existing stores]
└── styles/
    └── themes.css              # 7 themes as CSS custom properties
```

---

## Implementation Order (for inference/testing)

```
Week 1-2: Phase 1 — Sessions + Chat + Streaming
  → Target: hermes-webui parity (sessions, chat, profiles, skills, cron, memory)
  
Week 3-4: Phase 2 — Tiling WM + Keyboard Shortcuts
  → Target: hermes-web-computer tiling works with session UI
  
Week 5-6: Phase 3 — Docker/Observability + Config
  → Target: agent-os observability in Go+Svelte5
  
Week 7-8: Phase 4 — Command Palette + MCP + Voice
  → Target: All "nice to have" features from spec
  
Week 9-10: Phase 5 — Electron Shell
  → Target: Desktop app wrapping web UI
  
Week 11+: Phase 6 — Polish + Tests + CI/CD
  → Target: Shippable product
```

---

## Questions to Resolve Before Starting

1. **Database:** Use SQLite (modernc.org/sqlite) or JSON files? SQLite is faster for FTS but adds CGO-free dependency. JSON files (like hermes-webui) are simpler but slower with many sessions.

2. **Session storage location:** `~/.hermes/agent-os/` (default) or configurable via env?

3. **Electron shell:** Build as separate repo (`agent-os-desktop`) or in-tree (`desktop/`)?

4. **hermes-webui compatibility:** Should Agent-OS be able to READ hermes-webui's session files? (For users migrating from hermes-webui to Agent-OS without losing history)

5. **Multi-user:** Start single-user (current spec) or design for multi-user from the start?

6. **MCP:** Use official MCP SDK or hand-rolled? Official is better but adds complexity.

---

*This document is the master plan. Each phase should be broken into PR-sized tasks before implementation begins.*