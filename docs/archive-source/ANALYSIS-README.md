# hermes-computer-planning

> Analysis of four computer-use repos against the Agent-OS v1.2 spec. Purpose: assess viability, identify gaps, and critique the "Lean & Powerful" specification.

## Repos Analyzed

| Repo | Stars | Language | License | Last Pushed | Relevance |
|------|-------|----------|---------|-------------|-----------|
| [`coder/coder-desktop-linux`](https://github.com/coder/coder-desktop-linux) | 4 | C# (.NET/Avalonia) | AGPL-3.0 | 2026-03-05 | Remote workspace connectivity |
| [`roguedev-ai/kasm-mcp-server-v2`](https://github.com/roguedev-ai/kasm-mcp-server-v2) | 3 | Python | MIT | 2025-09-12 | Kasm workspace MCP control |
| [`bytebot-ai/bytebot`](https://github.com/bytebot-ai/bytebot) | 11,003 | TypeScript | Apache-2.0 | 2025-09-12 | Full desktop AI agent |
| [`trycua/cua`](https://github.com/trycua/cua) | 15,833 | Python/Swift/TS | MIT | 2026-05-09 | Cross-OS computer-use infra |

---

## 1. `coder/coder-desktop-linux` — Remote Workspace Bridge

### What It Is
C# (.NET 8) + Avalonia desktop app providing VPN-like connectivity to Coder workspaces. Tray app, VPN service integration, file sync — the Linux slice of the broader Coder Desktop product.

### Architecture
```
[Linux Desktop] → [Avalonia Tray App] → [Coder Connect (VPN)] → [Remote Workspace]
                                      → [File Sync]
```

### Strengths
- **Avalonia** — mature cross-platform UI framework for Linux
- **VPN integration** — no port-forwarding needed, clean network model
- **File sync** — bidirectional workspace ↔ local file operations

### Weaknesses for v1.2
- **Wrong language**: C#/.NET — doesn't fit Go backend + SvelteKit
- **AGPL-3.0** — viral license, incompatible with commercial reuse
- **Tiny**: 4 stars, 2 contributors, last push 2 months ago
- **Narrow scope**: Only connectivity, not agent control or desktop automation
- **Depends on external Coder server** — not self-contained

### Verdict
**Not useful directly.** The VPN connectivity concept is interesting (remote workspace access without port-forwarding), but the implementation is wrong language, wrong license, too narrow. Build from scratch in Go with `nhooyr.io/websocket` + WireGuard/Tailscale.

**Takeaway:** The *idea* of workspace-as-local-network is worth adopting. The code is not.

---

## 2. `roguedev-ai/kasm-mcp-server-v2` — Kasm MCP Control Plane

### What It Is
MCP server enabling AI assistants to manage Kasm Workspaces (containerized desktops). 21 tools across session management, command/file ops, user management, and monitoring.

### Architecture
```
[AI Assistant] → [MCP Server (Python)] → [Kasm API] → [Containerized Desktop]
                    ├── create_session
                    ├── execute_command
                    ├── read/write_file
                    └── get_screenshot
```

### Strengths
- **21 well-defined tools** — comprehensive workspace management surface
- **MCP Roots security** — bounded file operations, explicit allowlists
- **MIT license** — free to use and modify
- **Clean API mapping** — Kasm REST → MCP tool interface

### Weaknesses for v1.2
- **MCP dependency** — v1.2 explicitly rejects MCP ("One wire. JSON-RPC multiplexes UI, agent tools, and audio. No MCP translation layer.")
- **Stale**: Last pushed Sept 2025, 8 months ago
- **Tiny**: 3 stars, single maintainer
- **Kasm-specific**: Tightly coupled to Kasm API, not portable
- **Python-based**: v1.2 wants Go backend

### Verdict
**Partially useful as reference.** The 21-tool surface maps well to what v1.2's PTY supervisor needs. But the MCP layer is explicitly rejected. The *tool taxonomy* is valuable — reimplement as native Go handlers behind the JSON-RPC multiplexer.

**Takeaway:** Steal the tool taxonomy, drop the MCP wrapper. Kasm as desktop-in-container is an alternative to Docker + Caddy.

---

## 3. `bytebot-ai/bytebot` — Self-Hosted Desktop Agent

### What It Is
NestJS-based AI agent operating inside a containerized Linux desktop (Ubuntu 22.04 + XFCE). Controls a full virtual desktop — opens apps, handles 2FA, reads PDFs, runs CLI tools. Supports Claude, OpenAI, Gemini via LiteLLM. Deployable via Docker Compose, Railway, or Helm.

### Architecture
```
[Next.js UI] → [NestJS Agent] → [Ubuntu XFCE Desktop (Docker)]
                  ├── LiteLLM (Claude/GPT/Gemini)
                  ├── Computer-use API (screenshot, click, type)
                  └── REST API (task creation, desktop control)
```

### Strengths
- **11K stars** — proven community traction
- **Full desktop control** — not browser-only, handles any desktop app
- **Takeover Mode** — human can intervene mid-task (maps to v1.2's `Shift+Space` interrupt)
- **LiteLLM integration** — 100+ provider support, including local Ollama
- **REST API** — programmatic task creation and desktop control
- **Apache-2.0** — permissive

### Weaknesses for v1.2
- **Heavy stack**: NestJS + Next.js + Ubuntu XFCE + Firefox + VS Code — opposite of "lean by default"
- **No sub-100ms interrupt**: Takeover Mode is manual UI intervention, not atomic checkpoint + PTY signal
- **No Go component**: Entirely Node.js, no goroutine-level PTY control
- **No WebSocket multiplexer**: Uses separate REST endpoints per function
- **No deterministic state routing**: Agent interprets prompts — no backend-owned layout tree
- **No voice integration**: Text-only agent
- **No tiered security model**: Relies on container isolation, not declarative YAML permissions

### Verdict
**Strong conceptual reference, wrong implementation.** Bytebot proves the "AI controls a full desktop" model works at scale. Takeover Mode is closest existing implementation to v1.2's interrupt. But its stack is the antithesis of v1.2's lean philosophy.

**Takeaway:** The *product model* is right (self-hosted desktop agent with human takeover). The *implementation* is wrong for v1.2. LiteLLM integration and REST API design are worth studying. Containerized desktop (Ubuntu + XFCE) is reusable.

---

## 4. `trycua/cua` — Cross-OS Computer-Use Infrastructure

### What It Is
Comprehensive infrastructure for building, benchmarking, and deploying computer-use agents across macOS, Linux, Windows, and Android. Five core components: Driver (background macOS automation), SDK & Sandboxes (unified Python API), CuaBot (co-op CLI), Bench (evaluation), and Lume (Apple Silicon VMs).

### Architecture
```
[Cua SDK (Python)] → [Sandbox (Linux/macOS/Windows/Android)]
                        ├── Shell execution
                        ├── Screenshot capture
                        ├── Mouse/keyboard control
                        └── Multi-touch gestures (mobile)
                    → [Cua Driver (macOS)] → Background app automation
                    → [CuaBot (CLI)] → Multi-agent sandbox routing
                    → [Cua-Bench] → Evaluation & RL environments
                    → [Lume] → Apple Silicon VM management
```

### Strengths
- **15.8K stars, 60 contributors** — massive, active project
- **Cross-platform**: Linux, macOS, Windows, Android
- **Background automation**: macOS driver works without stealing cursor/focus
- **H.265 streaming**: Native window streaming with shared clipboard & audio
- **Benchmark suite**: OSWorld, ScreenSpot, Windows Arena
- **MIT license** — permissive
- **Active**: Last pushed 2 days ago, 471 releases
- **Python SDK**: Clean async API (`Sandbox.ephemeral()`, `sb.screenshot()`, `sb.mouse.click()`)
- **BYOI support**: Bring-your-own-image (.qcow2, .iso)

### Weaknesses for v1.2
- **Python, not Go**: SDK is Python-based, v1.2 wants Go backend
- **MCP integration**: CuaBot supports MCP, but v1.2 rejects it
- **Heavy**: 207MB repo, complex multi-component architecture
- **Not designed for human-in-the-loop desktop**: Optimized for autonomous agents
- **No interrupt model**: No atomic checkpoint + PTY signal
- **No layout engine**: No tiling, no border states, no keyboard-centric UI
- **No voice integration**

### Verdict
**Most useful repo for v1.2, with significant adaptation needed.** Cua's sandbox abstraction (`Sandbox.shell`, `Sandbox.screenshot`, `Sandbox.mouse`, `Sandbox.keyboard`) is exactly the interface v1.2's PTY supervisor needs — but reimplemented in Go. H.265 streaming, benchmark suite, and cross-platform approach are all valuable. Background automation driver proves agent control doesn't require stealing focus — relevant for v1.2's border states.

**Takeaway:** Cua's SDK design is the blueprint for v1.2's PTY supervisor interface. Port to Go. Benchmark suite validates interrupt latency claims. Lume's Apple Silicon virtualization could replace Docker for macOS deployments.

---

## Cross-Repo Analysis Against v1.2 Spec

### v1.2 Principles vs. Repo Reality

| v1.2 Principle | coder-desktop | kasm-mcp | bytebot | trycua/cua |
|---|---|---|---|---|
| **Backend owns truth** | ❌ C# client VPN | ⚠️ MCP mediates | ⚠️ NestJS owns state | ⚠️ Python SDK mediates |
| **Interrupt <100ms** | ❌ No model | ❌ No model | ⚠️ Takeover (slow) | ❌ No model |
| **One wire (no MCP)** | ⚠️ VPN tunnel | ❌ MCP-centric | ⚠️ Separate REST | ⚠️ MCP + REST |
| **Lean by default** | ⚠️ .NET heavy | ✅ Small Python | ❌ Full Ubuntu + Node | ⚠️ Multi-component |
| **Go backend** | ❌ C# | ❌ Python | ❌ TypeScript | ❌ Python/Swift |
| **SvelteKit frontend** | ❌ Avalonia | ❌ No frontend | ⚠️ Next.js | ❌ No frontend |
| **Sub-5MB RSS** | ❌ .NET runtime | ✅ Python server | ❌ Node.js + desktop | ⚠️ Python SDK |
| **Native PTY routing** | ❌ VPN focus | ⚠️ Command exec | ⚠️ Desktop control | ✅ Shell execution |
| **Fun-Audio-Chat** | ❌ | ❌ | ❌ | ❌ |
| **SQLite + JSONL** | ❌ | ❌ | ⚠️ Has telemetry | ⚠️ Has benchmarks |

### What v1.2 Gets Right (Validated by These Repos)

1. **Containerized desktop environments** — Bytebot and Cua both prove this model works. v1.2's Docker + Caddy approach is sound.
2. **Agent-in-loop with human takeover** — Bytebot's Takeover Mode validates the concept. v1.2's sub-100ms interrupt is a significant improvement.
3. **Cross-platform sandbox abstraction** — Cua's unified API (shell, screenshot, input) is the right interface model.
4. **Tool taxonomy for workspace control** — Kasm MCP server's 21 tools map well to what an agent needs.
5. **Self-hosted, privacy-first** — All four repos validate this as a market need.

### What v1.2 Gets Wrong (Critiques from Repo Reality)

1. **"No MCP" is a risky bet** — Three of four repos use MCP. Bytebot uses REST (which v1.2 replaces with JSON-RPC), but Cua and Kasm both bet heavily on MCP. The MCP ecosystem is growing — rejecting it entirely may limit interoperability with Claude Code, Cursor, and other MCP-native tools.
2. **"Sub-100ms interrupt" is unproven** — No existing repo achieves this. Bytebot's Takeover Mode is the closest and it's orders of magnitude slower. v1.2's claim needs validation.
3. **Go for PTY routing** — While Go is lean, the PTY ecosystem is more mature in Rust. Go's `creack/pty` is stable but less battle-tested for agent workloads.
4. **SvelteKit over React** — Bytebot's Next.js approach has 11K users validating it. SvelteKit is leaner but unproven for this use case.
5. **"One wire" complexity** — Multiplexing UI, agent tools, and audio over a single WebSocket is elegant but adds implementation complexity. Bytebot's separate endpoints are simpler to debug.
6. **Fun-Audio-Chat subprocess** — Spawning Python subprocesses for audio is fragile. Cua's native integration approach is more robust.
7. **Tiered YAML security** — Kasm MCP's Roots mechanism and Bytebot's container isolation are proven. v1.2's YAML-based permission model is untested.

### What v1.2 Should Borrow

| From | What to Borrow | Why |
|---|---|---|
| **trycua/cua** | Sandbox SDK interface (shell, screenshot, mouse, keyboard) | Clean, cross-platform API for PTY supervisor |
| **trycua/cua** | Benchmark suite (OSWorld, ScreenSpot) | Validate interrupt latency and agent performance |
| **trycua/cua** | H.265 streaming pattern | Efficient window streaming for UI deltas |
| **bytebot** | Takeover Mode UX pattern | Proven human-in-the-loop intervention model |
| **bytebot** | LiteLLM integration | 100+ provider support, including local models |
| **bytebot** | REST API design | Programmatic task creation and desktop control |
| **kasm-mcp-server-v2** | 21-tool taxonomy | Comprehensive workspace management surface |
| **kasm-mcp-server-v2** | MCP Roots security pattern | Bounded file operations with explicit allowlists |
| **coder-desktop-linux** | VPN connectivity concept | Seamless remote workspace access |

### What v1.2 Should Avoid

| From | What to Avoid | Why |
|---|---|---|
| **bytebot** | Full Ubuntu XFCE desktop | Too heavy — use minimal containers |
| **bytebot** | NestJS + Next.js stack | Node.js overhead contradicts "lean" |
| **coder-desktop-linux** | AGPL-3.0 license | Viral, incompatible with commercial reuse |
| **coder-desktop-linux** | C#/.NET stack | Wrong language, wrong ecosystem |
| **kasm-mcp-server-v2** | MCP dependency | Adds protocol tax, v1.2 rejects MCP |
| **kasm-mcp-server-v2** | Kasm-specific coupling | Not portable |
| **trycua/cua** | 207MB monorepo | Too complex — be modular |
| **trycua/cua** | Multi-component architecture | Driver + SDK + Bot + Bench + Lume is overkill |

---

## Revised Recommendations for v1.2

### Keep
- Go backend with `nhooyr.io/websocket`
- SvelteKit + Tailwind frontend
- Sub-100ms interrupt goal (validate with benchmarks)
- Backend-owned layout tree
- Docker Compose + Caddy deployment
- SQLite + JSONL telemetry

### Change
1. **Add MCP compatibility layer** — Don't reject MCP entirely. Support it as optional transport alongside JSON-RPC. Enables interoperability with Claude Code, Cursor, MCP-native tools.
2. **Adopt Cua's sandbox interface** — Port Python SDK to Go: `Sandbox.Shell()`, `Sandbox.Screenshot()`, `Sandbox.Mouse()`, `Sandbox.Keyboard()`.
3. **Use Bytebot's LiteLLM pattern** — Support 100+ providers out of the box, including local Ollama.
4. **Adopt Kasm's tool taxonomy** — 21 tools as native Go handlers.
5. **Add benchmark suite** — Use Cua-Bench patterns to validate interrupt latency, agent performance, state drift.
6. **Consider Rust for PTY routing** — If Go's `creack/pty` proves insufficient, pivot to Rust.

### Defer
- Fun-Audio-Chat subprocess — start text-only, add voice later
- H.265 streaming — WebSocket binary frames sufficient for initial release
- Cross-platform support — Linux first, macOS/Windows later
- BYOI support — pre-built images initially

---

## Implementation Priority (Revised)

| Week | Deliverable | Source Inspiration | Validation |
|------|-------------|-------------------|------------|
| 1 | Go WebSocket multiplexer + PTY supervisor (Cua sandbox interface) | trycua/cua SDK | `<50ms` layout delta, PTY execution |
| 2 | Svelte shell + layout tree + interrupt (Bytebot Takeover Mode) | bytebot Takeover Mode | `<100ms` interrupt, checkpoint restore |
| 3 | LiteLLM integration + tool taxonomy (Kasm 21 tools as Go handlers) | bytebot LiteLLM, kasm-mcp | 100+ providers, 21 tools functional |
| 4 | Docker Compose + Caddy + telemetry + benchmarks (Cua-Bench) | trycua/cua Bench | Single `docker compose up`, benchmark results |

---

## Final Assessment

The four repos validate v1.2's *product model* (self-hosted desktop agent with human-in-the-loop control) but expose significant *implementation risks*:

1. **MCP rejection is premature** — Ecosystem converging on MCP. Support it optionally.
2. **Sub-100ms interrupt is unproven** — Validate against Bytebot's Takeover Mode.
3. **Go for PTY routing is untested** — Rust has better PTY tooling for agent workloads.
4. **SvelteKit pivot is risky** — Bytebot's Next.js has 11K users validating it.
5. **Fun-Audio-Chat subprocess is fragile** — Native integration is more robust.

**v1.2's lean philosophy is correct.** But it should borrow the best patterns from these repos (Cua's SDK, Bytebot's Takeover Mode, Kasm's tool taxonomy, LiteLLM integration) while avoiding their bloat (full Ubuntu desktop, heavy Node.js, MCP-only).

**Recommendation:** Proceed with v1.2, but adopt the revised recommendations above. The spec is 80% right — the remaining 20% is learning from these repos' successes and failures.
