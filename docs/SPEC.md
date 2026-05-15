# Agent-OS: Browser-Native Collaborative Multi-Agent Environment

## System Specification Document v2.0

---

## 1. Executive Summary

Agent-OS is a browser-based, strictly tiled window management environment engineered for high-efficiency, keyboard-centric collaboration between a human developer, a text/terminal AI agent (Hermes), and a voice-native AI agent (Fun-Audio-Chat).

Built on a "Lean-First" philosophy, it relies on a high-performance Go backend and a lightweight Svelte 5 frontend. The system targets sub-100ms human override (Shift+Space), enforces strict backend-owned state synchronization, and provides a secure, containerized execution sandbox for autonomous operations.

> **Honest status:** This document describes the *aspirational* end state. The current implementation covers the core tiling layout, WebSocket multiplexer, PTY integration, and basic agent interface. Many features — particularly voice integration, Tier 1/2 security enforcement, and the MCP adapter — are planned but not yet implemented. Section 3 defines what is actually in flight.

---

## 2. Non-Goals

Explicitly **not** in scope for Agent-OS:

- **Floating windows** — strictly tiled, no drag-to-float
- **Multi-monitor** — single viewport, no window bridging across displays
- **Mobile/tablet** — keyboard-first desktop experience only
- **Native mobile companion app** — no iOS/Android clients
- **Real-time multi-user collaboration** (session sharing, shared cursors) — single user, single session per browser tab
- **Offline-first** — always-on connection to backend required
- **CRDT-based sync** — backend-owned truth, not peer-to-peer
- **AST-level code analysis** — security enforcement uses regex/allow-lists, not full language parsers
- **Hot-reload of running sessions** — session state survives backend restart; new sessions start clean
- **Third-party plugin ecosystem** — MCP adapter is internal/proprietary, not a public extension API

---

## 3. MVP Scope (v1.0 — What We Are Actually Building Now)

### In Flight (actively being implemented)

- [x] Go backend with WebSocket multiplexer (JSON-RPC envelopes)
- [x] Svelte 5 frontend with three-pane tiled layout
- [x] PTY integration (creack/pty) with ring buffer
- [x] Terminal tile (xterm.js)
- [x] File explorer (left panel, backend fs.list/fs.read)
- [x] Agent chat interface (right panel)
- [x] Shift+Space interrupt (SIGINT to PTY process group)
- [x] Basic session restore on reconnect
- [x] Tier 0 security (safe command allow-list)
- [ ] Monaco editor tile (lazy-loaded)
- [ ] Command palette (Ctrl+K)
- [ ] Keyboard shortcut system (Shift+Arrow, Shift+D, etc.)
- [ ] Resizable panel borders

### Planned v1.x (known gaps before 1.0)

- [ ] Tier 1/2 security enforcement UI (approval prompts, hard blocks)
- [ ] Docker sandbox (currently runs native PTY; containerized version planned)
- [ ] Layout persistence across backend restarts
- [ ] Basic observability: JSONL logging to disk

### Not in v1.0 (v2+)

- Voice / Fun-Audio-Chat integration
- MCP adapter
- LiteLLM proxy
- Playwright/headful browser automation
- Full-duplex voice with speech-detect interrupt
- Tiered security (beyond allow-list)

---

## 4. Technology Stack

### Backend

- **Language:** Go (net/http + nhooyr.io/websocket)
- **PTY:** creack/pty with 1MB ring buffer
- **Database:** modernc.org/sqlite (pure Go, no CGO)
- **Container runtime:** Docker client (for planned sandbox)
- **Target binary size:** <20MB compiled
- **Target backend RSS:** 30-80MB under normal load (Go runtime + PTY processes + SQLite page cache)

### Frontend

- **Framework:** Svelte 5 + Vite
- **Styling:** Tailwind CSS (utility-first)
- **Terminal:** xterm.js with WebGL addon (lazy-loaded)
- **Editor:** Monaco Editor (lazy-loaded)
- **Initial JS bundle (shell):** ~200KB gzipped (Svelte + xterm core + app bootstrap)
- **Full loaded (with Monaco):** ~800KB-1.2MB gzipped

### Metrics (Honest Targets)

| Metric | Target | Notes |
|--------|--------|-------|
| Backend binary size | <20MB | Compiled, static binary |
| Backend RSS | 30-80MB | Go runtime + PTY + SQLite under load |
| Initial JS (shell) | ~200KB gzip | App bootstrap, xterm core; Monaco lazy |
| Full loaded (Monaco) | ~1MB gzip | Editor tile loaded on demand |
| Interrupt latency | <150ms target, best-effort | Measured on localhost; p50 target |
| WS roundtrip (localhost) | 1-3ms | Direct loopback, no proxy |

---

## 5. Core UI Layout

### Three Panes

| Pane | Width | Content |
|------|-------|---------|
| Left (Explorer) | 280px, resizable | Tabbed: Files (tree), Apps (launcher) |
| Middle (Workspace) | 1fr | Binary tree of tiles; strictly tiled, no floating |
| Right (Agent) | 360px, resizable | Hermes chat, voice toggle, context history |

### Active Tile Visual State

The active tile has a glowing border, color-coded by execution state (set by backend):

- **Neon Blue:** Agent active, executing, or navigating
- **Amber/Pulsing:** Human has manual control (Shift+Space interrupt)
- **Red:** Execution blocked by security enforcer

### Bi-Directional Drag-and-Drop

- File from Left Panel → Right Panel: appends filepath to agent prompt queue
- Code block from Right Panel → Monaco (Middle): inserts at cursor

---

## 6. Keyboard Shortcuts

Intercepted at `window.addEventListener('keydown', fn, true)` before terminal TTY swallow.

| Shortcut | Action |
|----------|--------|
| `Shift+Space` | Universal interrupt (pause/resume agent) |
| `Ctrl+Alt+P` | Fallback interrupt (for IME/mobile) |
| `Shift+Arrow Keys` | Move focus between tiles |
| `Shift+D` | Toggle split orientation of active tile |
| `Shift+Alt+Arrow Keys` | Resize active tile borders |
| `Shift+F` | Toggle fullscreen for focused tile |
| `Shift+Q` | Kill/close active tile; rebalance |
| `Ctrl+K` | Open command palette |
| `Ctrl+B` | Toggle left panel |
| `Ctrl+Shift+B` | Toggle right panel |

---

## 7. Execution Sandbox & Security Model

### Tiered Security Enforcement

| Tier | Label | Examples | Behavior |
|------|-------|----------|----------|
| 0 | Safe | `ls`, `cat`, `go build`, `grep`, `find` | Executes immediately |
| 1 | Prompt | `git commit`, `docker build`, `rm` (non-recursive) | UI approval prompt; 30s JWT token on approval |
| 2 | Block | `rm -rf /`, `sudo`, `chmod 777 /` | Hard block; red tile border; no execution |

### Implementation Notes

- Tier classification uses regex + keyword allow-lists against the raw command string (not AST)
- Tier 1 approval generates a `security.approval.required` WS event; backend holds command until human approves or 60s timeout
- Path canonicalization: sandboxed process runs with `chroot` or Docker overlay; cannot escape `/agent/workspace/`
- Tier 2 block generates `security.blocked` event with reason

> **Known limitation:** Regex-based security is bypassable (e.g., `curl \| bash` sanitization has known gaps). Full protection requires process-level seccomp/i syscall filtering or a mandatory access control framework (AppArmor/SELinux). This is the correct long-term path; current tiered enforcement is a first-pass friction layer, not a hard security boundary.

---

## 8. Agent Lifecycle & Interrupt

### Interrupt Pipeline (Shift+Space)

1. **UI Freeze (optimistic):** Tile border turns amber immediately on keydown
2. **WS Signal:** Frontend sends `{"protocol":"agent","method":"interrupt"}` to backend
3. **OS Signal:** Backend sends `SIGINT` to PTY process group; if no response in 2s, sends `SIGTERM`
4. **Buffer Flush:** 1MB PTY ring buffer flushed to frontend
5. **Checkpoint:** State serialized to `~/.agent-os/checkpoints/<session_id>.json` (cursor offset, prompt stack, layout tree hash)

### Resume Policies

User-configurable, per-session:

| Policy | Behavior |
|--------|----------|
| **Stateless (A)** | Resume from exact instruction pointer; ignores human edits |
| **Diff-Aware (B)** | Compare Monaco buffer / DOM hash vs. pre-pause; adjust agent trajectory |
| **Confirm (C)** | Halt and prompt in Right Panel; user confirms or redirects |

Default: **Policy C** (most conservative). Selection made at session start or via command palette.

---

## 9. Protocol & Data Flow

### Single-Wire WebSocket Multiplexer

A single WS connection carries three independent protocol channels via envelope routing:

```json
{"protocol": "ui",   "method": "layout.split",    "params": {...}}
{"protocol": "agent", "method": "pty.write",        "params": {"pty_id": "...", "data": "ls\n"}}
{"protocol": "audio", "method": "audio.stream",     "params": {"opus_chunk": "..."}}
```

### Envelope Reference

**UI Protocol** — Layout operations, filesystem, app launch, focus state, panel toggle
**Agent Protocol** — PTY read/write, LLM prompts, tool results
**Audio Protocol** — Opus stream to/from Fun-Audio-Chat subprocess

### Progressive Enhancement

- **Default:** Proprietary JSON-RPC (minimal overhead, maximum speed)
- **Opt-in:** MCP Adapter for Claude Code / Cursor interop
- **Future:** LiteLLM proxy for multi-provider LLM routing

> **API Stability:** The JSON-RPC envelope format is **internal** and may change. No external consumers should depend on it. A public REST API for third-party integration is not in scope.

---

## 10. Voice & Audio Integration

### Planned Behavior (Not Yet Implemented)

- Frontend captures audio via `MediaRecorder` (Opus, 24kHz)
- Raw Opus chunks sent via `audio.stream` WebSocket envelope
- Backend relays to Fun-Audio-Chat Python subprocess
- Full-duplex: human speech detected → `0x03 0x02` (CONTROL PAUSE) frame sent to audio subprocess to halt inference
- Response streamed back as Opus and played via Web Audio API

### Zero MCP Tax

Voice input bypasses text tooling layers. Latency target (VAD detection to playback start): <500ms on localhost.

---

## 11. Failure Modes & Recovery

| Failure | Recovery |
|---------|----------|
| Backend crashes | Frontend detects WS disconnect; on reconnect, backend sends `layout.restore` with persisted layout tree |
| PTY process dies | Backend cleans up zombie; frontend shows "Process terminated" in tile; user can relaunch |
| Docker container killed | Sandbox is ephemeral by design; agent loses state; session checkpoint enables resume |
| Checkpoint corruption | If `checkpoint.json` is unreadable, start fresh session; log error |
| WS reconnect during active interrupt | Backend holds interrupt state for 30s; frontend retries; after timeout, abandons interrupt |
| Browser tab closed | Session persists in backend; re-opening same URL restores layout (session ID in localStorage) |

### Restart Semantics

- Backend restart: sessions survive (SQLite state + checkpoint files)
- Frontend restart (HMR): WS reconnects, no state loss
- Full browser close: session persists server-side; next visit with same session ID restores layout

---

## 12. Observability

### Local JSONL Logging

All agent actions, state transitions, and security decisions logged to:
```
~/.agent-os/logs/<session_id>.jsonl
```

Fields: `timestamp`, `session_id`, `event_type`, `actor` (human|agent|system), `details` (JSON).

### Metrics (Internal)

- Interrupt latency histogram (buckets: <50ms, <100ms, <200ms, <500ms, >500ms)
- WS message throughput per channel
- PTY buffer utilization
- Security: Tier 1 approval rate, Tier 2 block rate

### Cloud Sync (Future)

Async worker with exponential backoff syncs JSONL to Opik or Langfuse. Offline-capable: local buffer capped at 50MB, oldest entries evicted on overflow.

---

## 13. Architecture

```
┌──────────────────────────┐    WebSocket (JSON-RPC Multiplexer)    ┌─────────────────────────┐
│   Svelte 5 SPA            │ ◄────────────────────────────────────► │   Go Backend             │
│   (Frontend)              │  {"protocol":"ui|agent|audio", ...}    │   (Single Loop)         │
└──────────┬────────────────┘                                        └───────────┬─────────────┘
           │                                                                │
  ┌────────▼────────┐                                        ┌─────────────▼──────────────┐
  │ Layout Renderer │                                        │ PTY Supervisor             │
  │ Recursive Tiles │                                        │ (creack/pty, 1MB ring buf)  │
  └─────────────────┘                                        └─────────────┬──────────────┘
                                                                        │
                                                          ┌─────────────▼──────────────┐
                                                          │ Sandbox (Docker container)  │
                                                          │ or native PTY (MVP)        │
                                                          └─────────────────────────────┘
```

### Key Constraints

- Backend owns truth: frontend renders deltas, never calculates layout
- Single loop: all PTY, WS, audio, and security events on one goroutine
- No CRDTs: layout state is authoritative on backend only
- Sessions persist in SQLite; checkpoints on disk

---

## 14. Open Questions (Unresolved)

These need decisions before they can be spec'd:

1. **Session identity** — How is a session ID generated and when does it expire?
2. **Multi-session** — Can a user have multiple concurrent sessions? If so, how are they managed?
3. **Layout save/load** — Is there a named layout system (save layout as "debug workspace", restore later)?
4. **MCP adapter scope** — Which MCP tools are exposed? Just Hermes's built-in tools or also user-defined?
5. **Audio subprocess management** — How is Fun-Audio-Chat started/stopped? Managed by the Go backend or externally?
6. **Telemetry opt-in** — Is cloud sync enabled by default or explicit opt-in?
7. **Upgrade strategy** — How are backend updates handled without killing in-flight sessions?

---

## Appendix A: Reference Analysis (Archived → hermes-computer-planning)

**Source:** `ChonSong/hermes-computer-planning` — comparative analysis of 4 real-world repos against this spec.

**Repos analyzed:** coder-desktop-linux, kasm-mcp-server-v2, bytebot (11K stars), trycua/cua (15.8K stars)

**Key findings that shaped this spec:**

| Finding | Impact on v2.0 |
|---------|---------------|
| Sub-100ms interrupt unproven (Bytebot Takeover Mode is slowest) | Changed from hard requirement to `<150ms target, best-effort` |
| MCP ecosystem growing — rejection risky | Added optional MCP compatibility layer to Non-Goals / Protocol section |
| Cua's sandbox SDK (shell/screenshot/mouse) is exact interface needed | Validated PTY supervisor design |
| Kasm's 21-tool taxonomy covers workspace management surface | Informs planned v1.x tool implementation |
| Cua H.265 streaming is efficient but complex | Deferred to v2+; WebSocket binary frames sufficient for MVP |
| VPN connectivity concept from coder-desktop useful | Noted in analysis; not directly implemented |

**What was discarded:** Full Ubuntu/XFCE desktop (bytebot — too heavy), AGPL C# stack (coder-desktop — wrong license+language), MCP-only architecture (kasm — adds protocol tax).

**Archived repo:** `ChonSong/hermes-computer-planning` — branch `archive`, pushed May 2026.
