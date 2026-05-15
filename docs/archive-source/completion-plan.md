# Hermes-Web-Computer: Completion Plan

> From 70% working core to production-ready Agent-OS v1.2

**Generated:** 2026-05-11
**Current Status:** Core architecture proven (E2E test passes), integrations incomplete
**Target:** Production-ready tiling AI desktop

---

## 1. Current State Assessment

### ✅ Working Core (6 packages, ~1300 lines)

| Package | Status | Key Files | What Works |
|---------|--------|-----------|------------|
| **ws/multiplexer.go** | ✅ Complete | 470 lines | JSON-RPC routing (ui/agent/audio), session management, PTY lifecycle |
| **pty/supervisor.go** | ✅ Complete | Ring buffer + checkpoint | PTY start/stop, SIGINT handling, output streaming, 1MB ring buffer |
| **layout/tree.go** | ✅ Complete | Binary tree layout | split/mount/unmount/resize/swap/fullscreen ops, delta generation, SHA256 hashing |
| **security/security.go** | ✅ Complete | YAML permissions | Tier classification (safe/prompt/block), token-gated execution, default policy |
| **telemetry/telemetry.go** | ✅ Complete | JSONL ring buffer | Event writing, async cloud sync with exponential backoff, pruning |
| **audio/bridge.go** | ✅ Stub | Protocol relay | Connect/relay/interrupt skeleton (needs Fun-Audio-Chat protocol) |

### ✅ Frontend (Svelte 5 SPA)

| Component | Status | Key Features |
|-----------|--------|--------------|
| **App.svelte** | ✅ Working | WebSocket connection, layout rendering, Cmd+K palette, Cmd+? keymap |
| **Tile.svelte** | ✅ Working | Recursive layout renderer, keyboard ops (Shift+D/Q/F), focus management |
| **Terminal.svelte** | ✅ Working | xterm.js + FitAddon, PTY output streaming, resize observer |
| **CommandPalette.svelte** | ✅ Stub | UI exists but no commands wired |
| **KeymapOverlay.svelte** | ✅ Stub | UI exists but no keymap data |
| **Monaco.svelte** | ❌ Missing | File exists but not implemented |

### ✅ Infrastructure

| Component | Status | Details |
|-----------|--------|---------|
| **Docker Compose** | ✅ Working | agent-os + hermes + fun-audio + caddy |
| **Caddyfile** | ✅ Working | Reverse proxy, TLS, security headers |
| **Makefile** | ✅ Working | dev/build/test/clean targets |
| **CI (GitHub Actions)** | ✅ Working | Go build/vet/test + Node build |
| **E2E Test** | ✅ Passing | Connect → layout → PTY echo → interrupt → checkpoint |

### ❌ Incomplete / TODOs

| Item | Location | Priority | Effort |
|------|----------|----------|--------|
| Hermes agent integration | `multiplexer.go:tool.execute` | P0 | 2 days |
| LiteLLM adapter | New package | P0 | 3 days |
| Fun-Audio-Chat bridge | `bridge/audio_bridge.py` | P1 | 2 days |
| Monaco editor tile | `frontend/src/components/Monaco.svelte` | P1 | 1 day |
| Multi-user sessions | State management | P2 | 3 days |
| Computer-use sandbox | New package | P1 | 5 days |
| Vision testing | New infrastructure | P2 | 2 days |

---

## 2. Priority Order & Rationale

### P0: Make the Agent Actually Work

These are blocking — without them, the product doesn't function.

1. **Hermes Agent Integration** — Wire `tool.execute` to Hermes Agent's tool system
2. **LiteLLM Adapter** — Multi-provider model switching (MiniMax, OpenAI, Claude, etc.)

### P1: Complete the Vision

These make the product complete but don't block core functionality.

3. **Fun-Audio-Chat Bridge** — Full voice integration with protocol translation
4. **Monaco Editor Tile** — Code editing in the tiling interface
5. **Computer-Use Sandbox** — Containerized desktop for full desktop control

### P2: Polish & Scale

These improve the product but aren't required for launch.

6. **Multi-User Sessions** — Concurrent users with isolated workspaces
7. **Vision Testing** — Automated visual verification of the tiling UI

---

## 3. Detailed Task Breakdown

### P0-1: Hermes Agent Integration

**Goal:** Wire the existing `tool.execute` TODO to Hermes Agent's tool system.

**Current State:**
```go
// multiplexer.go:421
case "tool.execute":
    // TODO: execute tool via Hermes
    log.Printf("tool.execute: %s", string(env.Params))
```

**What needs to happen:**

1. **Create `backend/hermes/adapter.go`** — HTTP client to Hermes gateway
   - Connect to `http://host.docker.internal:8642` (or configurable URL)
   - Implement `ExecuteTool(toolName string, params map[string]interface{}) (result map[string]interface{}, err error)`
   - Implement `ListTools() ([]ToolDef, error)`
   - Implement `Chat(messages []Message) (response string, err error)`

2. **Wire adapter to multiplexer**
   - Initialize Hermes adapter in `NewMultiplexer()`
   - Replace `tool.execute` TODO with actual Hermes call
   - Route tool results back to client via WebSocket

3. **Update security enforcer**
   - Add tool execution to security tiers
   - Some tools should be "safe" (read-only), others "prompt" (require approval)

4. **Update telemetry**
   - Log tool execution events with tool name, params, result size, duration

**Interfaces:**
```go
type HermesAdapter struct {
    baseURL string
    client  *http.Client
    apiKey  string
}

func (h *HermesAdapter) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)
func (h *HermesAdapter) ListTools(ctx context.Context) ([]ToolDef, error)
func (h *HermesAdapter) Chat(ctx context.Context, messages []Message) (string, error)
```

**Estimated effort:** 2 days
**Dependencies:** Running Hermes Agent (already available at port 8642)

---

### P0-2: LiteLLM Adapter

**Goal:** Support 100+ LLM providers via LiteLLM pattern.

**Approach:** Two options:
- **Option A:** Use LiteLLM proxy server (Python) as sidecar
- **Option B:** Build Go adapter that speaks OpenAI-compatible API to multiple providers

**Recommendation: Option B** — Leaner, no Python dependency, fits Go backend philosophy.

**What needs to happen:**

1. **Create `backend/llm/provider.go`** — Provider abstraction
   ```go
   type Provider struct {
       Name       string
       BaseURL    string
       APIKey     string
       Model      string
       Client     *http.Client
   }
   
   func (p *Provider) Chat(ctx context.Context, messages []Message) (string, error)
   ```

2. **Create `backend/llm/router.go`** — Multi-provider router
   - Maintain list of configured providers
   - Route to default provider, fallback on error
   - Support model switching per session
   - Track usage/cost per provider

3. **Wire to Hermes adapter**
   - When Hermes doesn't have a tool for a request, route to LLM
   - LLM response flows back through WebSocket to client

4. **Configuration**
   - YAML config for providers:
     ```yaml
     providers:
       - name: minimax
         base_url: https://api.minimax.chat/v1
         model: MiniMax-M2.7
         api_key: ${MINIMAX_API_KEY}
       - name: openai
         base_url: https://api.openai.com/v1
         model: gpt-4o
         api_key: ${OPENAI_API_KEY}
     ```

**Estimated effort:** 3 days
**Dependencies:** None

---

### P1-1: Fun-Audio-Chat Bridge

**Goal:** Full voice integration with protocol translation.

**Current State:** `bridge/audio_bridge.py` is a stub. `backend/audio/bridge.go` has connect/relay/interrupt skeleton.

**What needs to happen:**

1. **Complete `bridge/audio_bridge.py`**
   - Connect to Fun-Audio-Chat on `ws://localhost:11235/api/chat`
   - Translate JSON-RPC envelopes to binary protocol:
     - `audio.stream` + opus_chunk → `0x01` + length + payload
     - `audio.interrupt` → `0x03` + `0x02` (PAUSE)
     - `audio.start` → `0x03` + `0x00` (START)
   - Stream audio back to Go via WebSocket
   - Handle heartbeat (30s ping)
   - Handle reconnect on disconnect

2. **Complete `backend/audio/bridge.go`**
   - Full protocol translation layer
   - Opus codec handling
   - Error handling and reconnection

3. **Update Docker Compose**
   - Add Fun-Audio-Chat service with GPU reservation
   - Configure audio bridge to connect

**Estimated effort:** 2 days
**Dependencies:** Fun-Audio-Chat server running

---

### P1-2: Monaco Editor Tile

**Goal:** Code editing tile in the tiling interface.

**What needs to happen:**

1. **Create `frontend/src/components/Monaco.svelte`**
   - Lazy-load Monaco editor WASM
   - Support multiple languages (TypeScript, Go, Python, etc.)
   - File path prop for context
   - Save shortcut (Cmd+S) → WebSocket message to backend

2. **Wire to layout engine**
   - When tile content is "monaco", render Monaco component
   - Support file path prop from layout tree

3. **Backend file operations** (optional, can defer)
   - Simple file read/write via PTY or dedicated endpoint

**Estimated effort:** 1 day
**Dependencies:** None

---

### P1-3: Computer-Use Sandbox

**Goal:** Containerized desktop environment for full desktop control.

**What needs to happen:**

1. **Create `backend/sandbox/manager.go`**
   - Docker API integration for sandbox container lifecycle
   - Screenshot capture via `docker exec` + `scrot` or `import`
   - Mouse/keyboard input routing via `xdotool`
   - Resource limits (memory, CPU, GPU)

2. **Create `backend/sandbox/container.go`**
   - Sandbox session management
   - WebSocket stream for screenshot updates
   - Input event routing

3. **Frontend sandbox viewer**
   - New tile type: "sandbox"
   - Screenshot stream (refresh every 500ms initially)
   - Mouse/keyboard overlay on screenshot

4. **Docker image**
   - Use bytebot's Ubuntu+XFCE image for MVP
   - Optimize to minimal container later

**Estimated effort:** 5 days
**Dependencies:** Docker API access

---

### P2-1: Multi-User Sessions

**Goal:** Concurrent users with isolated workspaces.

**What needs to happen:**

1. **Session isolation**
   - Each user gets unique session ID
   - Isolated PTY sessions
   - Isolated layout trees
   - Isolated security tokens

2. **WebSocket routing**
   - Route messages to correct session
   - Session cleanup on disconnect

3. **State management**
   - SQLite for session persistence
   - Session resume on reconnect

**Estimated effort:** 3 days
**Dependencies:** SQLite integration

---

### P2-2: Vision Testing

**Goal:** Automated visual verification of the tiling UI.

**What needs to happen:**

1. **Playwright test suite**
   - Take screenshots of each tile type
   - Verify layout correctness
   - Test keyboard operations

2. **Vision model integration**
   - Use Qwen3.6-plus (already configured) for image verification
   - Compare expected vs actual screenshots
   - Fail build on visual regression

3. **CI integration**
   - Run vision tests on every push to main
   - Cost: ~$0.01/screenshot × 20 screenshots = $0.20 per run

**Estimated effort:** 2 days
**Dependencies:** Qwen3.6-plus API access

---

## 4. Execution Timeline

| Week | Deliverables | Validation |
|------|-------------|------------|
| **Week 1** | P0-1: Hermes integration, P0-2: LiteLLM adapter | Agent responds to tool calls, model switching works |
| **Week 2** | P1-1: Audio bridge, P1-2: Monaco editor | Voice works, code editing tile functional |
| **Week 3** | P1-3: Computer-use sandbox | Desktop viewer works, mouse/keyboard routing |
| **Week 4** | P2-1: Multi-user, P2-2: Vision tests | Concurrent sessions, visual regression passing |

**Total:** 4 weeks to production-ready

---

## 5. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        hermes-web-computer                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────┐         WebSocket (JSON-RPC)          ┌─────────────┐│
│  │  Svelte 5    │ ◄───────────────────────────────────► │  Go Backend ││
│  │  SPA         │  {"protocol":"ui|agent|audio", ...}   │  Multiplexer││
│  │              │                                       └──────┬──────┘│
│  │ • Tile       │                                              │       │
│  │ • Terminal   │                    ┌─────────────────────────┼──┐    │
│  │ • Monaco     │                    │                         │  │    │
│  │ • CmdPalette │            ┌───────▼──────┐         ┌───────▼──▼──┐ │
│  │ • Keymap     │            │ PTY Supervisor│         │ Hermes Agent│ │
│  └──────────────┘            │ + Ring Buffer │         │ Integration │ │
│                              └──────────────┘         └──────┬──────┘ │
│                                                              │        │
│                              ┌───────────────────────────────┼──────┐ │
│                              │                               │      │ │
│                      ┌───────▼──────┐              ┌─────────▼────┐ │ │
│                      │ Layout Engine│              │ LiteLLM      │ │ │
│                      │ + Delta Ops  │              │ Router       │ │ │
│                      └──────────────┘              └──────────────┘ │ │
│                                                                     │ │
│                      ┌──────────────────────────────────────────┐  │ │
│                      │ Security Enforcer + Telemetry            │  │ │
│                      └──────────────────────────────────────────┘  │ │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ Sandbox Manager (Computer-Use)                               │   │
│  │ ┌────────────┐ ┌────────────┐ ┌────────────┐                │   │
│  │ │ Docker     │ │ Screenshot │ │ Input      │                │   │
│  │ │ Container  │ │ Capture    │ │ Routing    │                │   │
│  │ └────────────┘ └────────────┘ └────────────┘                │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ Audio Bridge (Fun-Audio-Chat)                                │   │
│  │ ┌────────────┐ ┌────────────┐ ┌────────────┐                │   │
│  │ │ Opus       │ │ Protocol   │ │ Reconnect  │                │   │
│  │ │ Codec      │ │ Translation│ │ Logic      │                │   │
│  │ └────────────┘ └────────────┘ └────────────┘                │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Hermes API changes | Low | High | Abstract adapter interface, version pinning |
| PTY stability issues | Medium | High | Ring buffer checkpoint, graceful degradation |
| LiteLLM provider downtime | Medium | Medium | Fallback chain, cached responses |
| Sandbox resource exhaustion | Low | High | Docker resource limits, auto-cleanup |
| Svelte 5 ecosystem immaturity | Low | Medium | Fallback to Svelte 4 if needed |
| Fun-Audio-Chat protocol changes | Low | Medium | Versioned protocol, compatibility layer |

---

## 7. Testing Strategy

### Unit Tests
- **Go:** `go test ./...` — All packages have test coverage
- **Svelte:** `npm test` — Component tests for Tile, Terminal

### Integration Tests
- **WebSocket:** Connect → send layout op → verify delta
- **PTY:** Start → write → read → interrupt → checkpoint
- **Hermes:** Execute tool → verify result
- **LiteLLM:** Chat with provider → verify response

### E2E Tests
- **Full flow:** Connect → layout → PTY → tool execute → LLM chat → interrupt
- **Vision:** Screenshot each page → verify with Qwen3.6-plus

### Performance Tests
- **Interrupt latency:** <100ms from Shift+Space to amber border
- **Layout render:** <50ms p99
- **WebSocket throughput:** >1000 messages/second

---

## 8. Deployment

### Development
```bash
make dev  # Backend + Frontend in parallel
```

### Production
```bash
docker compose up -d  # agent-os + hermes + fun-audio + caddy
```

### CI/CD
- GitHub Actions on push to main
- Go build/vet/test + Node build
- E2E test suite
- Vision tests (optional, configurable)

---

## 9. Success Metrics

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Interrupt latency | <100ms | Benchmark test |
| Layout render | <50ms p99 | Telemetry events |
| Agent response time | <2s | E2E test |
| Uptime | 99.9% | Health check monitoring |
| Memory usage | <50MB RSS | `ps` monitoring |
| Vision test pass rate | 100% | CI results |

---

## 10. Next Steps

1. **Start P0-1: Hermes Agent Integration** — This is the highest priority
2. **Parallel: P0-2: LiteLLM Adapter** — Can be done alongside Hermes integration
3. **Sequential: P1 features** — Audio, Monaco, Sandbox
4. **Polish: P2 features** — Multi-user, Vision tests

**Estimated total effort:** 4 weeks
**Estimated total cost:** ~$50 (vision testing API calls)

---

*Generated by autonomous agent exploration of hermes-web-computer codebase.*
*Based on: PLAN.md, README.md, docs/spec.md, and all source files.*
