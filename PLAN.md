# Hermes Web Computer v1.2 — Three-Column Layout Plan

## Architecture Overview

```
┌─────────────────┬──────────────────────────────┬──────────────────┐
│   LEFT COLUMN   │       MIDDLE COLUMN          │   RIGHT COLUMN   │
│   (280px)       │       (1fr, resizable)        │   (360px)        │
│                 │                              │                  │
│  [📁 Files]     │   ┌──────────┬──────────┐    │  ┌─────────────┐ │
│  [🚀 Apps]      │   │ Terminal │ Monaco   │    │  │ Agent Chat  │ │
│                 │   ├──────────┼──────────┤    │  │             │ │
│  File tree      │   │ Terminal │ Terminal │    │  │ History     │ │
│  or             │   └──────────┴──────────┘    │  │             │ │
│  App registry   │   Tiling area (2×2 max)      │  │ Input + 🎤  │ │
│                 │                              │  │             │ │
│                 │                              │  └─────────────┘ │
└─────────────────┴──────────────────────────────┴──────────────────┘
```

**Decisions resolved:**
- Layout: CSS grid, not binary tree (simpler, matches fixed-column spec)
- Middle column: keeps binary-tree tiling for 2×2 split apps
- Left column: tabbed (Files / Apps toggle)
- Backend: filesystem commands routed through existing WS multiplexer (one wire)
- Right column: text chat + voice toggle (Fun-Audio-Chat relay)
- Column widths: resizable via drag handles, persisted to localStorage
- App launcher: auto-discover from running sessions + quick-launch buttons

---

## Track 1: Backend — Filesystem API + Agent Protocol (est. 2h)

### 1.1 WS Multiplexer — New `ui` methods

Add handlers to `routeUI()` in `backend/ws/multiplexer.go`:

```go
case "fs.list":     // List directory → stream file entries
case "fs.read":     // Read file content
case "fs.write":    // Write file content
case "fs.stat":     // Stat a file/directory
case "fs.watch":    // Watch directory for changes (future)
case "apps.list":   // List available app types
case "apps.launch": // Launch a new app instance
```

Implementation:
- `fs.list` → `os.ReadDir`, return `{path, entries: [{name, type, size, modTime}]}`
- `fs.read` → `os.ReadFile`, return `{path, content (base64 for binary)}`
- `fs.write` → `os.WriteFile`, return `{path, bytes_written}`
- `fs.stat` → `os.Stat`, return `{path, exists, type, size, modTime}`
- `apps.list` → return `[{id: "terminal", name: "Terminal", icon: "⬛"}, {id: "editor", name: "Editor", icon: "📝"}, {id: "preview", name: "Preview", icon: "👁"}]`
- `apps.launch` → create PTY or editor tile, return `{tile_id, pty_id}`

Files to modify:
- `backend/ws/multiplexer.go` — add `routeFS()` and `routeApps()` methods
- `backend/pty/supervisor.go` — expose `ListSessions()` for app registry

### 1.2 Agent chat protocol

The `agent` protocol already exists. Extend it:

```
// Frontend → Backend
{protocol: "agent", method: "chat.send", params: {message: "...", context: {...}}}
{protocol: "agent", method: "voice.toggle", params: {enabled: true}}

// Backend → Frontend (streamed)
{protocol: "agent", event: "chat.reply", data: {message: "...", complete: true}}
{protocol: "agent", event: "voice.status", data: {enabled: true, recording: false}}
{protocol: "agent", event: "voice.transcript", data: {text: "..."}}
```

Backend acts as relay — agent chat messages go to the Hermes API, voice goes to Fun-Audio-Chat bridge.

Files to modify:
- `backend/ws/multiplexer.go` — `routeAgent()` handles new methods
- `backend/audio/bridge.go` — implement actual Fun-Audio-Chat connection

---

## Track 2: Frontend — Three-Column Layout + Components (est. 3h)

### 2.1 CSS Grid Shell

Replace `App.svelte` with:

```svelte
<div class="h-screen w-screen bg-gray-950 text-gray-100 grid"
     style="grid-template-columns: {leftW}px 1fr {rightW}px;">
  <LeftPanel />
  <MiddlePanel />
  <RightPanel />
  <ResizeHandle side="left" bind:width={leftW} />
  <ResizeHandle side="right" bind:width={rightW} />
</div>
```

New files:
- `frontend/src/components/LeftPanel.svelte` — tabbed sidebar
- `frontend/src/components/MiddlePanel.svelte` — tiling area
- `frontend/src/components/RightPanel.svelte` — agent chat
- `frontend/src/components/ResizeHandle.svelte` — drag-to-resize
- `frontend/src/components/FileTree.svelte` — directory listing
- `frontend/src/components/AppLauncher.svelte` — app registry + quick-launch
- `frontend/src/components/AgentChat.svelte` — chat history + input
- `frontend/src/components/VoiceToggle.svelte` — mic button + recording state

### 2.2 WS Store Extensions

Modify `frontend/src/stores/ws.ts`:

```typescript
// New stores
export const fileTree = writable<FileSystemEntry[]>([])
export const appRegistry = writable<AppType[]>([])
export const chatMessages = writable<ChatMessage[]>([])
export const voiceState = writable<{ enabled: boolean; recording: boolean }>({ enabled: false, recording: false })

// New WS methods
export function fsList(path: string): Promise<FileSystemEntry[]>
export function fsRead(path: string): Promise<string>
export function fsWrite(path: string, content: string): Promise<number>
export function launchApp(type: string): Promise<{tile_id: string}>
export function sendChat(message: string): void
export function toggleVoice(enabled: boolean): void
```

### 2.3 Component Details

**FileTree.svelte:**
- Sends `fs.list` on mount, renders nested tree
- Click file → opens in Monaco (new tile in middle column)
- Click folder → navigates into it
- Shows icons by extension
- Virtual scroll for large directories (1000+ files)

**AppLauncher.svelte:**
- Shows registered apps from `apps.list`
- Quick-launch: Terminal, Editor, Preview
- Running sessions list with re-connect buttons

**AgentChat.svelte:**
- Scrollable message history
- Input box with Enter to send, Shift+Enter for newline
- Voice toggle (mic icon, red when recording)
- Markdown rendering for agent responses
- Code blocks with copy button

**ResizeHandle.svelte:**
- 4px wide drag handle between columns
- Mouse down → track drag → update width
- Min widths: 200px left, 400px middle, 280px right
- localStorage persistence: `localStorage.setItem('ao-col-widths', JSON.stringify({left, right}))`

---

## Track 3: Backend Unit + Integration Tests (est. 1.5h)

### 3.1 Filesystem Handler Unit Tests

**File: `backend/ws/multiplexer_fs_test.go`**

Tests for each new FS handler:
- `TestFSList_Root` — List root directory, verify entries returned
- `TestFSList_Nested` — List nested subdirectory
- `TestFSList_NotFound` — Request non-existent path, verify error response
- `TestFSList_PermissionDenied` — Request restricted path (`/etc/shadow`), verify blocked
- `TestFSRead_TextFile` — Read text file, verify content matches
- `TestFSRead_BinaryFile` — Read binary file, verify base64 encoding
- `TestFSRead_NotFound` — Read non-existent file, verify error
- `TestFSWrite_Create` — Write new file, verify file exists on disk
- `TestFSWrite_Overwrite` — Overwrite existing file, verify content updated
- `TestFSWrite_DirTraversal` — Attempt `../../../etc/passwd`, verify blocked
- `TestFSStat_File` — Stat a file, verify size/type/modTime
- `TestFSStat_Dir` — Stat a directory, verify type=dir
- `TestFSStat_NotFound` — Stat non-existent, verify `exists: false`

### 3.2 App Protocol Unit Tests

**File: `backend/ws/multiplexer_apps_test.go`**

- `TestAppsList` — Verify terminal/editor/preview returned
- `TestAppsLaunch_Terminal` — Launch terminal, verify PTY session created, tile_id returned
- `TestAppsLaunch_Editor` — Launch editor, verify tile created
- `TestAppsLaunch_Invalid` — Launch unknown app type, verify error

### 3.3 WebSocket Integration Tests

**File: `backend/ws/integration_test.go`**

Spin up test server + WS client:
- `TestWSConnect` — Client connects, receives layout.initial
- `TestWSFSRoundTrip` — Send fs.list, receive entries, verify structure
- `TestWSFSWriteReadRoundTrip` — Write file via WS, read it back, verify content matches
- `TestWSAppLaunchAndPTY` — Launch terminal via WS, send command, verify output received
- `TestWSChatRoundTrip` — Send chat.send, verify agent protocol echo/relay
- `TestWSMultiProtocol` — Interleave ui/agent/audio messages, verify correct routing
- `TestWSReconnect_Resume` — Disconnect + reconnect, verify layout state restored (ResumePolicy B)

### 3.4 Security Enforcer Tests

**File: `backend/security/security_test.go`**

- `TestEnforcer_BlockDirTraversal` — Verify path traversal attacks blocked
- `TestEnforcer_BlockSymlinkEscape` — Verify symlink escapes blocked
- `TestEnforcer_TieredPermissions` — Verify tier config enforces command restrictions
- `TestEnforcer_MaxFileSize` — Verify large file reads capped
- `TestEnforcer_BlockedExtensions` — Verify `.exe`, `.bin`, etc. blocked from read/write

### 3.5 Layout Engine Tests

**File: `backend/layout/tree_test.go`**

- `TestSplit_ThenUnmount` — Split, mount tile, unmount, verify tree integrity
- `TestSplit_MaxDepth` — Split to depth 10, verify no panic
- `TestResize_Invalid` — Resize beyond bounds, verify clamped
- `TestSwap_Nodes` — Swap two leaf nodes, verify positions exchanged
- `TestDeltaGeneration` — Apply ops, verify SHA-256 hash matches expected
- `TestConcurrency` — Concurrent split/unmount ops, verify no race conditions

### 3.6 Benchmark Tests

**File: `backend/bench/fs_bench.go`**

- `BenchmarkFSList_LargeDir` — 10,000 file directory, measure serialization latency
- `BenchmarkFSRead_LargeFile` — 50MB file read + base64 encode, measure throughput
- `BenchmarkWSMessageRouting` — 10K messages/sec through multiplexer, measure p99 latency

---

## Track 4: Playwright + Vision Testing Framework (est. 3h)

### 4.1 Setup

```bash
cd /opt/data/hermes-web-computer
npm init -y --prefix e2e
cd e2e
npm install -D @playwright/test playwright axe-core @axe-core/playwright
npx playwright install --with-deps chromium
```

Config (`e2e/playwright.config.ts`):
```typescript
export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  timeout: 30000,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  use: {
    baseURL: 'http://localhost:3005',
    viewport: { width: 1440, height: 900 },
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  reporter: [['list'], ['html', { open: 'never' }], ['json', { outputFile: 'results.json' }]],
  webServer: {
    command: 'cd backend && go run ./cmd/server',
    port: 3005,
    reuseExistingServer: true,
    timeout: 15000,
  },
})
```

### 4.2 Functional E2E Test Suite

**Test 1: Layout Rendering** (`tests/01-layout.spec.ts`)
- Navigate to localhost:3005
- Assert three columns visible with correct ARIA labels
- Assert left panel has Files/Apps tabs (both clickable)
- Assert middle panel has initial tile with header
- Assert right panel has chat input + voice toggle
- Verify grid CSS computed styles match `280px 1fr 360px`
- **Vision check**: Screenshot, verify column proportions, no overflow, correct z-order

**Test 2: Column Resizing** (`tests/02-resize.spec.ts`)
- Drag left resize handle 100px wider
- Assert left column width = 380px (±2px tolerance)
- Assert middle column compressed proportionally
- Reload page → verify widths restored from localStorage
- **Edge cases:**
  - Drag to minimum width → verify clamped at 200px
  - Rapid drag (mouse move 50px in 10ms) → verify no layout jank
  - Drag beyond viewport → verify clamped to viewport bounds
- **Vision check**: Screenshot after each resize, verify no visual artifacts/overlaps/tearing

### 4.3 Complex Workflow Tests (`tests/workflows/`)

These test real multi-step user scenarios end-to-end, exercising the full stack.

**Test 3: File Edit Lifecycle** (`tests/workflows/file-edit.spec.ts`)
- Navigate to a directory, find `main.go`
- Click file → opens in Monaco tile in middle column
- Verify Monaco shows file content (line count matches)
- Edit: add `// test comment` at line 1
- Save via Ctrl+S → verify save confirmation toast
- Close tile → reopen same file
- Verify edited content persists (comment still there)
- Navigate to file tree → verify file icon shows modified state before save
- **Vision check**: Screenshot Monaco with code, verify syntax highlighting, line numbers, scroll

**Test 4: Multi-Terminal Pipeline** (`tests/workflows/pipeline.spec.ts`)
- Launch Terminal 1 → `mkdir -p /tmp/test-pipeline`
- Launch Terminal 2 → split from Terminal 1 (horizontal split)
- In Terminal 1: `echo '{"status":"ok"}' > /tmp/test-pipeline/config.json`
- In Terminal 2: `cat /tmp/test-pipeline/config.json` → verify JSON output
- Launch Terminal 3 → vertical split → 2×2 grid
- In Terminal 3: `echo "pipeline done" >> /tmp/test-pipeline/config.json`
- Close Terminal 2 → verify layout reflows to 2×1
- Close all → verify empty middle column shows "Launch an app" prompt
- **Vision check**: Screenshot 2×2 grid, verify each terminal independent, no output bleed

**Test 5: Chat + Context Workflow** (`tests/workflows/chat-context.spec.ts`)
- Open file in Monaco (e.g., `package.json`)
- Switch to agent chat, type: "What dependencies does this project have?"
- Verify agent receives file context (current open file path included)
- Verify response lists dependencies from package.json
- Send follow-up: "Add express" → verify agent understands context
- Type a code block in chat → verify markdown rendering with copy button
- Clear chat history → verify empty state
- Send 50 messages rapidly → verify queue processing, no dropped messages
- Reload → verify chat history restored from localStorage
- **Vision check**: Screenshot chat with code block, verify syntax highlighting, copy button, scroll

**Test 6: Full Session Recovery** (`tests/workflows/recovery.spec.ts`)
- Open file in Monaco, edit it (don't save)
- Launch a terminal, run `sleep 5` (long-running command)
- Send a chat message, wait for partial response
- Kill backend server → verify "Disconnected" state
- Verify unsaved editor state preserved in UI (dirty indicator)
- Restart server → verify auto-reconnect within 5s
- Verify editor still shows unsaved edits (client-side state preserved)
- Verify terminal shows reconnect message (PTY session may be lost — verify graceful handling)
- Verify chat history intact
- **Vision check**: Screenshot before/after disconnect, verify UI state transitions clean

**Test 7: Cross-Panel Interaction** (`tests/workflows/cross-panel.spec.ts`)
- Use Apps tab to launch terminal
- In terminal: `echo "hello from cli" > /tmp/cross-test.txt`
- Switch to Files tab → navigate to `/tmp` → verify `cross-test.txt` appears
- Click file → opens in new Monaco tile
- Verify content matches terminal output
- Edit file in Monaco: add second line
- Save → switch back to terminal → `cat /tmp/cross-test.txt` → verify both lines
- Toggle left panel off (Ctrl+B) → verify middle column expands to fill
- Toggle right panel off (Ctrl+Shift+B) → verify full-screen middle column
- Toggle both back on → verify original layout restored
- **Vision check**: Screenshot full-screen mode, verify no empty gaps, proper expansion

### 4.4 Resilience & Chaos Tests (`tests/chaos/`)

**Test 8: Server Death** (`tests/chaos/server-death.spec.ts`)
- Load page, verify connected
- Kill backend server (`SIGTERM`)
- Assert "Disconnected" state visible within 3s
- Verify reconnection attempt every 2s (assert 3+ attempts in 10s)
- Restart server → verify auto-reconnect within 5s
- Verify layout state restored (ResumePolicy B)

**Test 9: Network Throttling** (`tests/chaos/network.spec.ts`)
- Apply `Slow 3G` throttling via CDP
- Load page → measure time to interactive (< 8s on 3G)
- Send chat message → verify loading indicator, eventual response
- Drag resize handle → verify smooth (60fps) despite network throttling
- Apply `Offline` mode → verify disconnected state, cached UI still renders

**Test 10: WebSocket Flood** (`tests/chaos/ws-flood.spec.ts`)
- Send 1000 WS messages/sec for 10 seconds
- Verify no messages dropped (count sent = count received)
- Verify memory stable (heap snapshot before/after, < 10% growth)
- Verify UI responsive (measure frame rate, > 30fps during flood)

**Test 11: Concurrent Sessions** (`tests/chaos/concurrent.spec.ts`)
- Open 5 browser tabs, each connecting to same WS endpoint
- Each tab launches a terminal, sends `echo tab-{N}`
- Verify each tab receives its own PTY output (no cross-talk)
- Kill one tab → verify other 4 unaffected

### 4.5 Accessibility Tests (`tests/a11y/`)

**Test 12: Keyboard Navigation** (`tests/a11y/keyboard.spec.ts`)
- Tab through entire UI → verify focus ring visible on every interactive element
- Tab order: left panel → middle → right panel (logical flow)
- Arrow keys navigate file tree
- Escape closes panels/palettes
- Ctrl+K opens command palette
- Enter activates focused element
- **Verify**: No keyboard traps (can always tab out)

**Test 13: Screen Reader** (`tests/a11y/screen-reader.spec.ts`)
- All interactive elements have `aria-label` or `aria-labelledby`
- File tree uses `role="tree"` with proper `aria-expanded` on folders
- Chat messages use `role="log"` with `aria-live="polite"`
- Terminal output uses `role="log"` with `aria-live="off"` (don't read every char)
- Resize handles use `role="separator"` with `aria-valuemin/max/now`
- **Axe-core scan**: Zero violations (critical/serious), max 2 minor

**Test 14: Color Contrast** (`tests/a11y/contrast.spec.ts`)
- All text meets WCAG AA (4.5:1 for normal, 3:1 for large)
- Interactive elements meet WCAG AA (3:1 against adjacent colors)
- Red/green states distinguishable by more than color (icon + text)
- Focus indicators meet 3:1 contrast against unfocused state

### 4.6 Visual Regression Tests (`tests/visual/`)

**Test 15: Baseline Screenshots** (`tests/visual/baseline.spec.ts`)
- Run once to generate baseline screenshots for each view:
  - `baseline-default.png` — 1440×900, all columns
  - `baseline-resized.png` — left 400px, right 450px
  - `baseline-files-open.png` — Files tab, folder expanded
  - `baseline-apps-open.png` — Apps tab with running sessions
  - `baseline-chat-active.png` — Chat with 3 messages
  - `baseline-voice-recording.png` — Voice toggle in recording state

**Test 16: Pixel Diff Regression** (`tests/visual/regression.spec.ts`)
- On each run, compare against baselines
- Fail if pixel diff > 0.1% (tolerance for anti-aliasing)
- Generate diff images showing changed regions
- Store new screenshots as `current/` for manual review

### 4.7 Performance Tests (`tests/perf/`)

**Test 17: Load Time** (`tests/perf/load-time.spec.ts`)
- Measure Time to First Byte (TTFB) < 100ms
- Measure DOM Content Loaded < 500ms
- Measure Time to Interactive < 1s
- Measure WS connection established < 2s
- Bundle size: `vite build` output < 400KB (current ~348KB, budget 400KB)

**Test 18: Memory Leak Detection** (`tests/perf/memory.spec.ts`)
- Take heap snapshot after page load
- Run 50 interactions (open/close files, launch/close terminals, send chat messages)
- Take heap snapshot after
- Verify JS heap growth < 5MB (no leaked closures/event listeners)
- Verify DOM node count stable (no detached nodes)

**Test 19: Long-Running Stability** (`tests/perf/stability.spec.ts`)
- Run app for 30 minutes (simulated)
- Send WS heartbeat every 30s
- Verify no memory growth > 20MB
- Verify WS connection stable (no unexpected disconnects)
- Verify PTY output buffer doesn't grow unbounded (check backend metrics endpoint)

### 4.7 Vision Analysis Script

**File: `e2e/scripts/vision_analyze.py`**

For each test screenshot, runs automated visual QA:

```python
# Checks performed:
# 1. Layout proportions: columns at expected widths (±5px)
# 2. Text readability: no overlapping text, minimum 12px font
# 3. Visual hierarchy: headers larger/bolder than body
# 4. Responsiveness: no horizontal scroll at any viewport
# 5. WCAG contrast: pixel-level contrast ratio sampling
# 6. State visibility: active/focus/hover states visually distinct
# 7. Animation quality: no jank during resize (frame analysis)
# 8. Color consistency: no unexpected color shifts between views
```

Uses:
- `Pillow` for pixel analysis (contrast, color, layout detection)
- `numpy` for histogram analysis (detect text overlap, blank regions)
- `colorthief` for palette extraction (verify design consistency)
- Reports pass/fail per criterion with evidence (annotated screenshots)

### 4.9 Test Execution Matrix

| Test Suite | Browsers | CI | Local | Frequency |
|-----------|----------|----|----|-----------|
| Functional (1-2) | Chromium only | ✅ | ✅ | Every commit |
| Workflows (3-7) | Chromium only | ✅ | ✅ | Every commit |
| Chaos (8-11) | Chromium only | ✅ | ✅ | Nightly |
| Accessibility (12-14) | Chromium only | ✅ | ✅ | Every commit |
| Visual Regression (15-16) | Chromium only | ✅ | ✅ | Every commit |
| Performance (17-19) | Chromium only | ✅ | ✅ | Nightly |

---

## Track 5: CI/CD Integration (est. 1.5h)

### 5.1 GitHub Actions Workflow

**File: `.github/workflows/ci.yml`**

```yaml
name: CI
on: [push, pull_request]
jobs:
  # Phase 1: Fast checks
  lint-and-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Go build + vet + test
        run: cd backend && go build ./... && go vet ./... && go test ./... -v
      - name: Frontend build
        run: cd frontend && npm ci && npm run build
      - name: Bundle size check
        run: |
          SIZE=$(du -sk frontend/dist/ | cut -f1)
          [ $SIZE -lt 400 ] || { echo "Bundle > 400KB!"; exit 1; }

  # Phase 2: E2E tests (runs only if Phase 1 passes)
  e2e-functional:
    needs: lint-and-build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go + Node
        uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - uses: actions/setup-node@v4
        with: { node-version: '22' }
      - name: Install Playwright
        run: cd e2e && npm ci && npx playwright install --with-deps chromium
      - name: Run E2E tests
        run: cd e2e && npx playwright test tests/01-*.spec.ts tests/02-*.spec.ts tests/03-*.spec.ts tests/04-*.spec.ts tests/05-*.spec.ts tests/06-*.spec.ts
      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with: { name: e2e-results, path: e2e/playwright-report/ }

  # Phase 3: Accessibility (runs in parallel with E2E)
  a11y:
    needs: lint-and-build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run a11y tests
        run: cd e2e && npm ci && npx playwright install --with-deps chromium && npx playwright test tests/a11y/

  # Phase 4: Nightly (chaos + perf + vision)
  nightly:
    runs-on: ubuntu-latest
    if: github.event_name == 'schedule' || github.ref == 'refs/heads/main'
    schedule:
      - cron: '0 3 * * *'  # 3am UTC
    steps:
      - uses: actions/checkout@v4
      - name: Run full test suite
        run: cd e2e && npm ci && npx playwright install --with-deps && npx playwright test
      - name: Run vision analysis
        run: cd e2e && python3 scripts/vision_analyze.py
      - name: Run backend benchmarks
        run: cd backend && go test ./bench -bench=. -benchmem -benchtime=10s
```

### 5.2 Pre-commit Hooks

**File: `.husky/pre-commit`** (or `.githooks/pre-commit`):
```bash
#!/bin/sh
set -e
cd backend && go vet ./... && go test ./... -count=1
cd ../frontend && npm run build
cd ../e2e && npx playwright test tests/01-layout.spec.ts
```

### 5.3 Makefile Updates

```makefile
.PHONY: dev build test test-backend test-e2e test-a11y test-visual test-perf test-chaos test-all vision ci clean

dev:
	cd backend && go run ./cmd/server &
	cd frontend && npm run dev

build:
	cd backend && go build -o agent-os ./cmd/server
	cd frontend && npm run build

test: test-backend test-e2e

test-backend:
	cd backend && go test ./... -v -count=1
	cd backend && go test ./bench -bench=. -benchmem -benchtime=5s

test-e2e:
	cd e2e && npx playwright test tests/01-*.spec.ts tests/02-*.spec.ts tests/03-*.spec.ts tests/04-*.spec.ts tests/05-*.spec.ts tests/06-*.spec.ts

test-a11y:
	cd e2e && npx playwright test tests/a11y/

test-visual:
	cd e2e && npx playwright test tests/visual/

test-perf:
	cd e2e && npx playwright test tests/perf/

test-chaos:
	cd e2e && npx playwright test tests/chaos/

test-all: test-backend test-e2e test-a11y test-visual test-perf

vision: test-all
	cd e2e && python3 scripts/vision_analyze.py

ci: test-all vision

clean:
	rm -rf backend/agent-os frontend/dist e2e/playwright-report e2e/test-results
```

## Track 6: Integration & Polish (est. 1h)

### 6.1 State Synchronization
- File tree updates when files are modified in middle column tiles
- Chat messages persist across page reloads (localStorage)
- Column widths persist across reloads
- Running sessions listed in app launcher

### 6.2 Keyboard Shortcuts
- `Ctrl+K` — Command palette (existing)
- `Ctrl+?` — Keymap overlay (existing)
- `Ctrl+B` — Toggle left panel
- `Ctrl+Shift+B` — Toggle right panel
- `Ctrl+Enter` — Send chat message
- `Ctrl+Shift+N` — New terminal
- `Escape` — Close panels/palettes

### 6.3 Makefile Updates

(Already covered in Track 5.3 — this section is just the integration layer.)

---

## Execution Order

```
Phase 1 (Track 1 + 2a): Backend FS API + CSS Grid Shell
  └─ Add fs.list/read/write/stat to multiplexer
  └─ Replace App.svelte with 3-column grid
  └─ Create LeftPanel, MiddlePanel, RightPanel shells

Phase 2 (Track 2b + 2c): FileTree + AppLauncher + AgentChat
  └─ Wire FileTree to ws.fsList
  └─ Wire AppLauncher to ws.apps.list/launch
  └─ Wire AgentChat to ws.agent.chat.send
  └─ Implement ResizeHandle with localStorage

Phase 3 (Track 3): Backend Unit + Integration Tests
  └─ Write filesystem handler unit tests (14 tests)
  └─ Write app protocol unit tests (4 tests)
  └─ Write WebSocket integration tests (7 tests)
  └─ Write security enforcer tests (5 tests)
  └─ Write layout engine tests (6 tests)
  └─ Write benchmark tests (3 tests)
  └─ Run: go test ./... -count=1 → fix failures

Phase 4 (Track 4): Playwright + Vision Testing
  └─ Setup Playwright (Chromium only)
  └─ Write 2 functional E2E tests
  └─ Write 5 complex workflow tests (multi-step scenarios)
  └─ Write 4 chaos/resilience tests
  └─ Write 3 accessibility tests
  └─ Write 2 visual regression tests
  └─ Write 3 performance tests
  └─ Write vision analysis script (Python/Pillow/numpy)
  └─ Run full suite, fix failures

Phase 5 (Track 5): CI/CD Integration
  └─ GitHub Actions: lint+build → e2e → a11y → nightly
  └─ Pre-commit hooks
  └─ Makefile targets for all test suites

Phase 6 (Track 6): Integration & Polish
  └─ Keyboard shortcuts
  └─ State synchronization
  └─ Final full test suite + vision pass
```

---

## File Change Summary

| File | Action | Reason |
|------|--------|--------|
| `backend/ws/multiplexer.go` | Modify | Add fs.list/read/write/stat, apps.list/launch |
| `backend/ws/multiplexer_fs_test.go` | Create | Filesystem handler unit tests |
| `backend/ws/multiplexer_apps_test.go` | Create | App protocol unit tests |
| `backend/ws/integration_test.go` | Create | WebSocket integration tests |
| `backend/security/security_test.go` | Create/Update | Security enforcer tests |
| `backend/layout/tree_test.go` | Create/Update | Layout engine tests |
| `backend/bench/fs_bench.go` | Create | Filesystem benchmarks |
| `backend/audio/bridge.go` | Modify | Implement Fun-Audio-Chat relay |
| `frontend/src/App.svelte` | Rewrite | 3-column CSS grid shell |
| `frontend/src/stores/ws.ts` | Modify | Add fileTree, appRegistry, chatMessages stores + methods |
| `frontend/src/components/LeftPanel.svelte` | Create | Tabbed sidebar |
| `frontend/src/components/MiddlePanel.svelte` | Create | Tiling area container |
| `frontend/src/components/RightPanel.svelte` | Create | Agent chat panel |
| `frontend/src/components/ResizeHandle.svelte` | Create | Drag-to-resize |
| `frontend/src/components/FileTree.svelte` | Create | Directory listing |
| `frontend/src/components/AppLauncher.svelte` | Create | App registry + quick-launch |
| `frontend/src/components/AgentChat.svelte` | Create | Chat history + input |
| `frontend/src/components/VoiceToggle.svelte` | Create | Mic button + recording state |
| `e2e/package.json` | Create | E2E test dependencies |
| `e2e/playwright.config.ts` | Create | Playwright config (Chromium only) |
| `e2e/tests/01-layout.spec.ts` | Create | Layout rendering test |
| `e2e/tests/02-resize.spec.ts` | Create | Column resizing test + edge cases |
| `e2e/tests/workflows/file-edit.spec.ts` | Create | File edit lifecycle (open→edit→save→reopen) |
| `e2e/tests/workflows/pipeline.spec.ts` | Create | Multi-terminal pipeline (split→write→read→close) |
| `e2e/tests/workflows/chat-context.spec.ts` | Create | Chat with file context, markdown, queue |
| `e2e/tests/workflows/recovery.spec.ts` | Create | Full session recovery (crash→reconnect) |
| `e2e/tests/workflows/cross-panel.spec.ts` | Create | Cross-panel interaction (terminal→file→edit→save) |
| `e2e/tests/chaos/server-death.spec.ts` | Create | Server death resilience test |
| `e2e/tests/chaos/network.spec.ts` | Create | Network throttling test |
| `e2e/tests/chaos/ws-flood.spec.ts` | Create | WebSocket flood test |
| `e2e/tests/chaos/concurrent.spec.ts` | Create | Concurrent sessions test |
| `e2e/tests/a11y/keyboard.spec.ts` | Create | Keyboard navigation test |
| `e2e/tests/a11y/screen-reader.spec.ts` | Create | Screen reader / ARIA test |
| `e2e/tests/a11y/contrast.spec.ts` | Create | Color contrast test |
| `e2e/tests/visual/baseline.spec.ts` | Create | Baseline screenshot generator |
| `e2e/tests/visual/regression.spec.ts` | Create | Pixel diff regression test |
| `e2e/tests/perf/load-time.spec.ts` | Create | Load time performance test |
| `e2e/tests/perf/memory.spec.ts` | Create | Memory leak detection test |
| `e2e/tests/perf/stability.spec.ts` | Create | Long-running stability test |
| `e2e/scripts/vision_analyze.py` | Create | Vision analysis runner (Pillow/numpy) |
| `.github/workflows/ci.yml` | Create | GitHub Actions CI pipeline |
| `Makefile` | Modify | Add all test suite targets |

---

## Success Criteria

### Build
1. `go build ./...` — zero errors, zero warnings
2. `go vet ./...` — zero issues
3. `vite build` — bundle < 400KB (current budget: 400KB)
4. `npx playwright install --with-deps` — all 3 browsers install cleanly

### Backend Tests
5. 45+ Go unit + integration tests — all pass, 0 flaky
6. Benchmarks: FS list 10K files < 50ms, WS routing p99 < 5ms
7. Security: all path traversal, symlink escape, permission tests pass

### E2E Tests
8. 2 functional tests — pass on Chromium
9. 5 workflow tests — multi-step scenarios covering file lifecycle, terminal pipeline, chat context, session recovery, cross-panel interaction
10. 4 chaos tests — server death recovery < 5s, no message loss during flood
11. 3 accessibility tests — zero axe-core critical/serious violations, keyboard navigation complete
12. 2 visual regression tests — pixel diff < 0.1% on all baselines
13. 3 performance tests — TTI < 1s, heap growth < 5MB, 30-min stability < 20MB growth

### Vision Analysis
14. 8 visual criteria pass on all screenshots: proportions, readability, hierarchy, responsiveness, contrast, state visibility, animation quality, color consistency

### CI/CD
15. GitHub Actions: lint+build → e2e → a11y pass on every push
16. Nightly: chaos + perf + vision + benchmarks pass on main branch
17. Pre-commit hook: blocks commit if backend tests or build fail

### Feature Completeness
18. Three-column layout renders at 1440×900 and 1920×1080
19. File manager navigates, opens files in middle column
20. App launcher launches terminal/editor, shows running sessions
21. Agent chat sends/receives messages, voice toggle works
22. Column resizing works, widths persist to localStorage
23. Keyboard shortcuts functional (Ctrl+K, Ctrl+B, Ctrl+Shift+N, etc.)
