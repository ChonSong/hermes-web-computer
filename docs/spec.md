# Agent-OS v1.2: Complete Specification

**Status:** Production-Ready | **Philosophy:** Lean but Powerful

## Key Decisions

- Go backend: `net/http` + `nhooyr.io/websocket` (no framework)
- Frontend: Vanilla Svelte 5 + Vite (no SvelteKit)
- SQLite: `modernc.org/sqlite` (pure Go, no CGO)
- PTY: `creack/pty` (maintained, battle-tested)
- Audio: WebSocket relay to Fun-Audio-Chat server on localhost:11235
- Protocol: Custom JSON-RPC envelope with protocol tags
- Deploy: Docker Compose + Caddy

## Architecture

```
┌─────────────────┐    WebSocket (JSON-RPC Multiplexer)    ┌─────────────────┐
│   SvelteKit UI  │ ◄───────────────────────────────────► │   Go Backend    │
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
