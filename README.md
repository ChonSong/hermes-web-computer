# hermes-web-computer

> Agent-OS v1.2: browser-native, strictly tiled, keyboard-centric collaborative environment for a human developer, Hermes (text/terminal agent), and Fun-Audio-Chat (voice agent).

## Philosophy

**Lean but Powerful.** No Temporal, no CRDTs, no AST parsers, no heavy telemetry sync. Backend-owned truth, sub-100ms interrupt, zero protocol bloat.

## Architecture

```
┌─────────────────┐    WebSocket (JSON-RPC Multiplexer)    ┌─────────────────┐
│   Svelte 5 SPA  │ ◄───────────────────────────────────► │   Go Backend    │
│ (Capture Phase) │  {"protocol":"ui|agent|audio", ...}   │  (Single Loop)  │
└────────┬────────┘                                       └────────┬────────┘
         │                                                         │
   ┌─────▼─────┐                                           ┌───────▼────────┐
   │ Layout    │                                           │ PTY Supervisor │
   │ Renderer  │                                           │ Cgroups+PID NS │
   └───────────┘                                           └───────┬────────┘
                                                                   │
                                                          ┌────────▼────────┐
                                                          │ Hermes / Audio  │
                                                          │ Docker+Subproc  │
                                                          └─────────────────┘
```

## Monorepo Structure

```
├── backend/          # Go: multiplexer, PTY, state, security, audio, telemetry
│   ├── cmd/server/   # Entry point
│   ├── ws/           # WebSocket multiplexer + JSON-RPC routing
│   ├── pty/          # PTY supervisor + ring buffer + checkpoint
│   ├── state/        # Layout tree + session state + checkpoints
│   ├── security/     # YAML permissions + token-gated execution
│   ├── audio/        # Fun-Audio-Chat WebSocket relay
│   └── telemetry/    # JSONL ring buffer + async cloud sync
├── frontend/         # Svelte 5 + Vite + xterm + monaco
│   ├── src/
│   │   ├── stores/   # WebSocket store + layout state
│   │   └── components/ # Tile, Terminal, Monaco, CommandPalette
│   └── dist/         # Built static files (served by Go/Caddy)
├── deploy/           # Docker Compose, Caddyfile
├── bridge/           # Python: Fun-Audio-Chat subprocess wrapper
├── bench/            # Go: interrupt latency benchmark harness
└── docs/             # Spec, architecture diagrams, decisions log
```

## Quick Start

```bash
# Backend
cd backend && go mod tidy && go run cmd/server/main.go

# Frontend
cd frontend && npm install && npm run dev
```

## Tech Stack

| Layer | Choice | Why |
|-------|--------|-----|
| Frontend | Svelte 5 + Vite + Tailwind | <50KB initial bundle, zero VDOM |
| Backend | Go (`net/http` + `nhooyr.io/websocket`) | Sub-5MB RSS, native goroutines |
| PTY | `creack/pty` | Battle-tested, maintained |
| Storage | `modernc.org/sqlite` | Pure Go, no CGO |
| Deploy | Docker Compose + Caddy | Auto-TLS, WebSocket proxy |
| Audio | Fun-Audio-Chat (WebSocket relay) | Native Opus, no MCP tax |

## Key Principles

1. **Backend owns truth** — client renders deltas, zero layout drift
2. **Interrupt <100ms** — `Shift+Space` triggers optimistic UI freeze + atomic checkpoint
3. **One wire** — JSON-RPC multiplexes UI, agent tools, and audio
4. **Lean by default** — cut Temporal, CRDTs, AST parsers; opt-in only
5. **Voice-native** — Fun-Audio-Chat direct Opus stream, full-duplex interrupt

## Spec

Full specification: [`docs/spec.md`](docs/spec.md)

## License

MIT
