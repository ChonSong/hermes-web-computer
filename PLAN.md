# hermes-web-computer — Implementation Plan

**Goal:** Complete Agent-OS v1.2 from the working core (6 commits, ~1300 lines) to production-ready.
**Approach:** 6 parallel tracks, each owned by one subagent. Merge via PR.

---

## Track 1: Go Layout Engine (`backend/layout/`)

**Owner:** Backend agent
**Files to create:**
- `backend/layout/tree.go` — Binary tree layout, split/resize/merge ops
- `backend/layout/delta.go` — Diff computation between tree versions
- `backend/layout/state.go` — Session state with layout version + hash

**Interfaces to implement:**
```go
type LayoutTree struct {
    ID        string       `json:"id"`
    Type      string       `json:"type"`       // "split" or "leaf"
    Direction string       `json:"direction"`  // "h" or "v"
    Content   string       `json:"content"`    // "xterm", "monaco", "welcome"
    Path      string       `json:"path"`       // file path for monaco
    PTYID     string       `json:"pty_id"`     // PTY session ID for xterm
    Children  []LayoutTree `json:"children"`
}

type LayoutOp struct {
    Op        string `json:"op"`          // "split", "mount", "unmount", "resize", "swap"
    TargetID  string `json:"target_id"`
    Direction string `json:"direction,omitempty"`
    Content   string `json:"content,omitempty"`
    Size      float64 `json:"size,omitempty"`  // 0.0-1.0 relative size
}

func (t *LayoutTree) Apply(op LayoutOp) (delta []LayoutOp, err error)
func (t *LayoutTree) Hash() string  // sha256 of canonical JSON
func DiffTree(old, new LayoutTree) []LayoutOp
```

**Keyboard ops to route:**
- `Shift+D` → swap split orientation of active tile
- `Shift+Alt+Arrow` → resize tile borders
- `Shift+F` → toggle fullscreen
- `Shift+Q` → close tile (merge with sibling)
- `Shift+Arrows` → move focus between tiles

**Dependencies:** `crypto/sha256`, `encoding/json`
**No external deps.**

---

## Track 2: Go Security Engine (`backend/security/`)

**Owner:** Backend agent
**Files to update/replace:**
- `backend/security/security.go` — Full implementation (currently stub)
- `backend/security/yaml.go` — YAML config parser
- `backend/security/enforcer.go` — Command classification + token generation

**Interfaces to implement:**
```go
type SecurityConfig struct {
    Tiers map[string]TierConfig `yaml:"tiers"`
}

type TierConfig struct {
    Paths []string `yaml:"paths"`
    Cmds  []string `yaml:"cmds"`
}

type Enforcer struct {
    config SecurityConfig
    tokens map[string]ExecToken  // token -> command + expiry
}

func (e *Enforcer) Classify(cmd string, cwd string) (tier string, err error)
func (e *Enforcer) GrantToken(cmd string) (token string, expiry int64)
func (e *Enforcer) ValidateToken(token string, cmd string) (bool, error)
func (e *Enforcer) LoadConfig(path string) error
```

**Matching rules:**
- `path.Match` for glob patterns
- Exact match for `cmds` (no regex, keep it lean)
- Token expiry: 30s default

**Dependencies:** `path.Match`, `crypto/rand` for token generation
**Optional:** `gopkg.in/yaml.v3` for YAML parsing (add to go.mod)

---

## Track 3: Go Telemetry Engine (`backend/telemetry/`)

**Owner:** Backend agent
**Files to update/replace:**
- `backend/telemetry/telemetry.go` — Full implementation (currently stub)
- `backend/telemetry/sync.go` — Async cloud sync worker

**Interfaces to implement:**
```go
type Event struct {
    Ts         int64   `json:"ts"`
    SessionID  string  `json:"session"`
    Type       string  `json:"event"`
    User       string  `json:"user,omitempty"`
    Policy     string  `json:"policy,omitempty"`
    DriftScore float64 `json:"drift_score,omitempty"`
    Command    string  `json:"cmd,omitempty"`
    Token      string  `json:"token,omitempty"`
    Outcome    string  `json:"outcome,omitempty"`
    Tool       string  `json:"tool,omitempty"`
    Path       string  `json:"path,omitempty"`
    Size       int     `json:"size,omitempty"`
}

type RingBuffer struct {
    file   *os.File
    maxMB  int
    mu     sync.Mutex
}

func (rb *RingBuffer) Write(event Event) error
func (rb *RingBuffer) SyncToCloud(endpoint string) error  // POST to Langfuse/Opik
func (rb *RingBuffer) Prune() error  // Remove oldest events when > maxMB
```

**Sync logic:**
- Background goroutine ticks every 60s
- Reads buffered events, POSTs to configured endpoint
- Exponential backoff on failure (1s, 2s, 4s, 8s, max 60s)
- Prunes oldest events when file > maxMB

**Dependencies:** `encoding/json`, `net/http`, `os`
**No external deps.**

---

## Track 4: Frontend Tile Engine (`frontend/src/components/`)

**Owner:** Frontend agent
**Files to create:**
- `frontend/src/components/Tile.svelte` — Recursive layout renderer
- `frontend/src/components/CommandPalette.svelte` — Ctrl/Cmd+K overlay
- `frontend/src/components/KeymapOverlay.svelte` — Ctrl/Cmd+? help
- `frontend/src/components/Monaco.svelte` — Code editor (lazy-loaded)

**Tile component spec:**
```svelte
<!-- Recursive: renders itself for children -->
<script>
  export let node: LayoutTree
  export let focused: boolean
  // Max depth 3, lazy-mount Monaco/Xterm on viewport entry
</script>

{#if node.type === 'split'}
  <div class="flex {node.direction === 'h' ? 'flex-row' : 'flex-col'}">
    {#each node.children as child}
      <Tile {child} focused={focusedChildId === child.id} />
    {/each}
  </div>
{:else}
  <div class="border-2 {focused ? 'border-blue-500' : 'border-gray-700'}">
    {#if node.content === 'xterm'}
      <Terminal ptyId={node.pty_id} />
    {:else if node.content === 'monaco'}
      <Monaco path={node.path} />
    {:else}
      <Welcome />
    {/if}
  </div>
{/if}
```

**Store updates needed (`stores/ws.ts`):**
- Add `focusChange` handler
- Add `layoutMutation` send method
- Add keyboard handlers for layout ops

**Dependencies:** `@monaco-editor/react` or `monaco-editor` (WASM)

---

## Track 5: Python Audio Bridge (`bridge/`)

**Owner:** Python agent
**Files to update/replace:**
- `bridge/audio_bridge.py` — Full Fun-Audio-Chat protocol translation (currently stub)

**Protocol translation table:**
| Go JSON-RPC | Fun-Audio-Chat Binary | Direction |
|---|---|---|
| `audio.stream` + opus_chunk | `0x01` + length + payload | Go → FA |
| `audio.interrupt` | `0x03` + `0x02` (PAUSE) | Go → FA |
| `audio.start` | `0x03` + `0x00` (START) | Go → FA |
| FA audio response | `audio.response` event | FA → Go |
| FA text output | `audio.text` event | FA → Go |
| FA tool_calls | `audio.tool_calls` event | FA → Go |

**Implementation:**
- Connect to Fun-Audio-Chat on `ws://localhost:11235/api/chat`
- Translate JSON-RPC envelopes to binary protocol
- Stream audio back to Go via WebSocket
- Handle heartbeat (30s ping)
- Handle reconnect on disconnect

**Dependencies:** `websockets`, `asyncio`, `struct`, `logging`

---

## Track 6: Integration & Deploy

**Owner:** DevOps agent
**Files to update/create:**
- `backend/go.sum` — Run `go mod tidy` (add yaml v3 dep)
- `backend/cmd/server/main.go` — Wire all packages together
- `backend/ws/multiplexer.go` — Wire layout + security + telemetry handlers
- `deploy/docker-compose.yml` — Add Fun-Audio-Chat service
- `deploy/Caddyfile` — Update for production
- `.github/workflows/ci.yml` — Go build + test + frontend build
- `Makefile` — `make dev`, `make build`, `make test`

**Integration points:**
- Multiplexer routes `ui/layout.*` → Layout engine
- Multiplexer routes `agent/pty.write` → PTY supervisor (wired)
- Multiplexer routes `agent/tool.*` → Security enforcer → PTY
- Multiplexer routes `audio/*` → Audio bridge
- Telemetry records all events
- Layout state synced via `layout.initial` + `layout.delta` events

---

## Validation Criteria

| Track | Metric | Target |
|---|---|---|
| Layout Engine | p99 layout render | <50ms |
| Security | Blocked `rm -rf` halts + red border | Works |
| Telemetry | Events written to JSONL + synced | Works |
| Tile Engine | Recursive layout, keyboard nav, border states | Works |
| Audio Bridge | Opus relay + interrupt | Works |
| Integration | `docker compose up` → full flow | Works |

---

## Execution Order

**Phase 1 (parallel):** All 6 tracks run simultaneously
**Phase 2:** Integration — wire everything in `main.go` + `multiplexer.go`
**Phase 3:** Test — `go test ./...` + `npm test` + interrupt latency benchmark
