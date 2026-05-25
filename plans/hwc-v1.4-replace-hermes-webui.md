# Blueprint: HWC v1.4 — Replace Hermes WebUI + Migrate from agent-os

**Created**: 2026-05-25
**Status**: ✅ v1.4 complete (Phase 0-5 all done)
**Project**: hermes-web-computer (ChonSong/hermes-web-computer)
**Inspiration repos**: hermes-workspace (outsourc-e/hermes-workspace), features-list (ChonSong/features-list), agent-os (ChonSong/agent-os), hermes-webui (ChonSong/hermes-webui)

---

## Overview

Hermes WebUI (Python/vanilla JS, v0.51.54, ~5303 tests) is being replaced by HWC (Go+Svelte5 tiling desktop). All Hermes WebUI features must be present in HWC. agent-os (Express/React/Postgres) is on the chopping block — its container management and nanobot integration must migrate to HWC's Go backend. No external dependencies beyond Docker. Self-contained with Hermes integration.

---

## Dependency Graph

```
Phase 0 (Critical Fixes)
 ├── Step 0.1: Fix WS connection (frontend :3113 → :3005)
 └── Step 0.2: Wire all Go managers to multiplexer (main.go bootstrap)

Phase 1 (Core Wiring)
 ├── Step 1.1: Docker UI — DockerPanel + container lifecycle (start/stop/remove/logs)
 ├── Step 1.2: Session store path → ~/.hermes/hermes-web-computer/sessions/
 └── Step 1.3: Security config path → ~/.hermes/hermes-web-computer/security.yaml

Phase 2 (Hermes WebUI Feature Parity)
 ├── Step 2.1: Workspace file browser (wire FileTree to fs.list/fs.read/fs.write)
 ├── Step 2.2: Slash command registry (/ prefix, autocomplete, command palette)
 ├── Step 2.3: File upload (drag-drop → fs.write)
 ├── Step 2.4: Session search + rename/delete dialogs
 └── Step 2.5: Context meter (token tracking in chat)

Phase 3 (agent-os Feature Migration)
 ├── Step 3.1: Container create (docker run / docker compose up)
 ├── Step 3.2: Container remove with confirmation
 ├── Step 3.3: Image management (list, inspect, remove)
 └── Step 3.4: Compose project support (docker compose ls/ps/stop)

Phase 4 (Missing Features from features-list)
 ├── Step 4.1: Research cards (markdown rendering of links/search results)
 ├── Step 4.2: Connection status handling (reconnect on drop)
 ├── Step 4.3: Message search within session
 └── Step 4.4: Session project grouping + color coding

Phase 5 (Xpra Integration)
 ├── Step 5.1: xpra/manager.go stub → real implementation (Xvfb + unix socket)
 ├── Step 5.2: XpraTile.svelte (HTML5 client iframe)
 └── Step 5.3: SSH tunnel support for remote xpra

Phase 6 (Observability Expansion)
 ├── Step 6.1: Real trace view (structured logging from telemetry)
 ├── Step 6.2: Cost ledger (per-session LLM cost tracking)
 └── Step 6.3: Skills usage analytics

Phase 7 (Multi-User / Team Support — later)
 ├── Step 7.1: OIDC auth (Keycloak token validation)
 ├── Step 7.2: Per-user session store isolation
 └── Step 7.3: Coder workspace lifecycle (provision/suspend/delete)

Testing Gate (after each phase):
  → go build && npm run build && playwright test
  → No regressions on existing tiles
```

---

## Steps

### Step 0.1: Fix WebSocket Connection Bug
**Status**: pending
**Dependencies**: none
**Estimated**: 15 min

**Context**: The frontend at `frontend/src/stores/ws.ts:139` hardcodes `ws://localhost:3113/ws` — this connects to the agent-os Express backend, NOT the HWC Go backend on port 3005. The HWC Go backend is running but idle. This is the single most critical bug — nothing else works until this is fixed.

**Tasks**:
- [ ] Edit `frontend/src/stores/ws.ts` — change default URL from `ws://localhost:3113/ws` to `ws://localhost:3005/ws`
- [ ] Edit `frontend/vite.config.ts` — add proxy for `/api` → `http://localhost:3005` (if not already present)
- [ ] Verify: `curl -s http://localhost:3005/api/system/info` returns HWC info (not 404)

**Verification**:
```bash
curl -s http://localhost:3005/api/system/info | python3 -m json.tool
# Should return {"version":"v1.3.0",...}
```

**Exit criteria**: Frontend WS connects to HWC Go backend on :3005. Chat panel sends messages to HWC multiplexer, not agent-os.

---

### Step 0.2: Wire All Go Managers in main.go Bootstrap
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 30 min

**Context**: The multiplexer has `SetDockerManager()`, `SetSessionStore()`, `SetConfigManager()`, `SetAudioBridge()` methods, but `main.go` needs to call them during startup. Currently the docker manager may not be attached, making all `docker.*` WS methods no-ops. Need to verify and fix the bootstrap sequence.

**Tasks**:
- [ ] Find and read `cmd/server/main.go` or the entry point in `backend/`
- [ ] Verify `dockerMgr := docker.NewManager()` is called and `multiplexer.SetDockerManager(dockerMgr)` is called
- [ ] Verify `sessionStore` is created and `multiplexer.SetSessionStore(store)` is called
- [ ] Verify `configMgr` is created and `multiplexer.SetConfigManager(configMgr)` is called
- [ ] Check that `hermesURL` env var (`HERMES_API_URL`) defaults to `http://localhost:8642` (Hermes agent)

**Verification**:
```bash
# After startup, test docker methods
echo '{"protocol":"ui","method":"docker.list","id":"test","ts":1}' | \
 websocat ws://localhost:3005/ws --text || true
# Should return docker.list.response, not docker.error

go test ./backend/... -count=1 -timeout=60s
```

**Exit criteria**: All managers attached, docker.list returns real containers, session store initialized at correct path.

---

### Step 1.1: DockerPanel — Container Lifecycle UI
**Status**: pending
**Dependencies**: Step 0.2
**Estimated**: 2 hours

**Context**: HWC has `docker.Manager` in Go with `ListContainers`, `StartContainer`, `StopContainer`, `RemoveContainer`, `GetStats`, `GetLogs`. Frontend has store functions (`dockerList()`, `dockerContainerStats()`, `dockerContainerLogs()`) but no DockerPanel component. Need to build a real container management UI that shows running/stopped containers, stats, logs, and allows lifecycle operations.

**Tasks**:
- [ ] Create `frontend/src/components/DockerPanel.svelte`:
  - Container list table (ID, name, image, state, status, created)
  - Click row → expand to show stats (CPU%, mem used/limit, network I/O)
  - Action buttons: Start / Stop / Restart / Remove
  - Log viewer (tail -f via websocket stream)
  - Search/filter containers by name or state
- [ ] Add `DockerPanel` to RightPanel tab bar (add "Containers" tab)
- [ ] Wire `dockerList()` → on mount, poll every 30s
- [ ] Wire action buttons to `docker.start/stop/restart/remove` WS methods
- [ ] Style to match HWC dark theme (#191919 base, consistent with other panels)

**Verification**:
```bash
npm run build
# Open HWC → right panel → Containers tab
# Verify: real running containers listed, stats updating, start/stop works
playwright test tests/docker*.spec.ts  # if test file exists
```

**Exit criteria**: Real container management UI working. Can list, start, stop, remove containers from HWC UI.

---

### Step 1.2: Session Store Path Migration
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 15 min

**Context**: Session store currently initialized at `~/.agent-os` (from multiplexer.go line 159). This should migrate to `~/.hermes/hermes-web-computer/` to keep HWC state self-contained.

**Tasks**:
- [ ] Update `backend/ws/multiplexer.go` — change default session store path from `~/.agent-os` to `~/.hermes/hermes-web-computer`
- [ ] Update security config path similarly
- [ ] Move existing sessions from `~/.agent-os/sessions/` to new location if upgrading existing install

**Verification**:
```bash
ls ~/.hermes/hermes-web-computer/sessions/  # should exist after first run
cat ~/.hermes/hermes-web-computer/sessions/*.json | python3 -m json.tool | head  # session files readable
```

**Exit criteria**: All session data stored under `~/.hermes/hermes-web-computer/`, not `~/.agent-os/`.

---

### Step 1.3: Security Config Path Migration
**Status**: pending
**Dependencies**: Step 1.2
**Estimated**: 10 min

**Context**: Security enforcer loads from `~/.agent-os/security.yaml` (multiplexer.go line 146). Should be under HWC's state directory.

**Tasks**:
- [ ] Update `security.Enforcer.LoadConfig()` path in multiplexer.go
- [ ] Create default security config at new location if not found
- [ ] Document security tier structure in `docs/SECURITY.md`

**Verification**:
```bash
cat ~/.hermes/hermes-web-computer/security.yaml  # should be valid YAML
```

**Exit criteria**: Security config under HWC state dir. Default permissive tier if no config.

---

### Step 2.1: Workspace File Browser — Wire FileTree to FS Methods
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 2 hours

**Context**: `FileTree.svelte` exists but is a stub. Hermes WebUI has a full workspace file browser (in `static/workspace.js`) with list_dir, read_file_content, inline preview (images, markdown, PDFs), file create/delete/rename. HWC needs this wired to the Go `fs.list/fs.read/fs.write/fs.stat/fs.rename/fs.delete` handlers in the multiplexer.

**Tasks**:
- [ ] Review `static/workspace.js` in hermes-webui for the full feature set
- [ ] Wire `FileTree.svelte` to `fs.list` (list directory entries)
- [ ] Wire click to `fs.read` (read file content)
- [ ] Wire right-click context menu → `fs.rename`, `fs.delete`, `fs.write` (for edits)
- [ ] Add file preview panel (markdown render, image display, PDF viewer)
- [ ] Add breadcrumb navigation
- [ ] Implement drag-drop file upload → `fs.write`

**Verification**:
```bash
npm run build
# Navigate to workspace → file tree shows real files
# Click file → content displayed in preview
# Right-click → rename/delete options work
```

**Exit criteria**: Full workspace file browser in HWC, functionally equivalent to Hermes WebUI's.

---

### Step 2.2: Slash Command Registry
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 1.5 hours

**Context**: Hermes WebUI has `commands.js` (1302 lines) with a `/` prefix command registry, parser, and autocomplete dropdown. HWC has CommandPalette.svelte but no slash command support.

**Tasks**:
- [ ] Read `static/commands.js` in hermes-webui to understand the command format
- [ ] Create `frontend/src/stores/commands.svelte.ts` — command registry store
- [ ] Implement `parseCommand(input: string)` → `{cmd: string, args: string[]}`
- [ ] Implement autocomplete dropdown (show matching commands as user types `/`)
- [ ] Wire chat composer to detect `/` prefix and show autocomplete
- [ ] Add built-in commands: `/session`, `/model`, `/workspace`, `/skills`, `/clear`, `/help`
- [ ] Commands fire WS events to multiplexer for handling

**Verification**:
```bash
npm run build
# In chat input, type "/" → autocomplete dropdown appears
# Select command → executes correctly
```

**Exit criteria**: Slash command registry functional. Autocomplete on `/`. Built-in commands execute.

---

### Step 2.3: File Upload (Drag-Drop)
**Status**: pending
**Dependencies**: Step 2.1
**Estimated**: 1 hour

**Context**: Hermes WebUI supports drag-drop file upload via `upload.py` (multipart parser). Files can be attached to messages and written to workspace.

**Tasks**:
- [ ] Add `fs.write` handler in multiplexer to accept base64-encoded file content
- [ ] Create `frontend/src/components/FileUpload.svelte` (drag-drop zone overlay)
- [ ] Wire drag-drop onto chat panel → `fs.write` → attach path to message
- [ ] Support multiple file uploads

**Verification**:
```bash
# Drag a file onto HWC chat panel
# File appears as attachment in composer
# On send, file content written to workspace
```

**Exit criteria**: Drag-drop file upload works. Files written to workspace, paths attached to messages.

---

### Step 2.4: Session Search + Rename/Delete
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 1 hour

**Context**: SessionsPanel currently shows session list. Need search, rename, delete dialogs.

**Tasks**:
- [ ] Add search input to SessionsPanel
- [ ] Implement session search via `session.search(query)` → `sessionStore.Search(query)`
- [ ] Add right-click context menu: Rename, Delete, Archive, Duplicate
- [ ] Add confirm dialog for Delete/Archive
- [ ] Wire `session.rename` and `session.delete` WS methods in multiplexer

**Verification**:
```bash
npm run build
# Sessions panel → search box → type query → results filter
# Right-click session → Delete → confirm → session removed
```

**Exit criteria**: Session search + rename/delete functional.

---

### Step 2.5: Context Meter (Token Tracking)
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 45 min

**Context**: Hermes WebUI has a context bar showing token usage. HWC should show context utilization in chat header.

**Tasks**:
- [ ] Add token counting logic (estimate from message content lengths)
- [ ] Show context meter in ChatPanel header (like context-bar in features-list)
- [ ] Color-code: green (<60%), yellow (60-85%), red (>85%)
- [ ] On overflow warning, suggest context compression

**Verification**:
```bash
npm run build
# Open chat → header shows token meter
# Long conversation → meter updates, color changes
```

**Exit criteria**: Context meter visible in chat, updates live, color-coded correctly.

---

### Step 3.1: Container Create
**Status**: pending
**Dependencies**: Step 1.1
**Estimated**: 1.5 hours

**Context**: agent-os has container management for nanobot. HWC docker.Manager has `CreateContainer` in the CLI wrapper. Need to expose container creation in the DockerPanel UI.

**Tasks**:
- [ ] Add "Create" button to DockerPanel
- [ ] Dialog with: image name, container name, port mappings, env vars, volume mounts
- [ ] Wire to `docker.create()` in Go manager
- [ ] Show created container in list immediately

**Verification**:
```bash
npm run build
# DockerPanel → Create → fill form → submit
# Container appears in list, starts successfully
```

**Exit criteria**: Container create dialog works. Can create containers from UI.

---

### Step 3.2: Container Remove with Confirmation
**Status**: pending
**Dependencies**: Step 1.1
**Estimated**: 30 min

**Context**: Remove button needs confirmation dialog (stop containers first, then remove).

**Tasks**:
- [ ] Add confirm dialog: "Stop and remove container X?"
- [ ] If container running → stop first, then remove
- [ ] Handle force remove (stopped containers with `force=true`)
- [ ] Show toast notification on success/error

**Verification**:
```bash
# Right-click container → Remove → confirm
# Container removed from list
```

**Exit criteria**: Remove with confirmation. Running containers stopped before removal.

---

### Step 3.3: Image Management
**Status**: pending
**Dependencies**: Step 1.1
**Estimated**: 1 hour

**Context**: No image management UI. Users need to see what images are available and remove unused ones.

**Tasks**:
- [ ] Add "Images" tab in DockerPanel
- [ ] List images: repository, tag, size, created
- [ ] Inspect image (show layers, config)
- [ ] Remove unused images
- [ ] Pull image by name

**Verification**:
```bash
npm run build
# DockerPanel → Images tab
# List shows real images, remove works
```

**Exit criteria**: Image management tab functional.

---

### Step 3.4: Compose Project Support
**Status**: pending
**Dependencies**: Step 3.1
**Estimated**: 1 hour

**Context**: agent-os uses `docker compose` for nanobot. Need to support compose projects in DockerPanel.

**Tasks**:
- [ ] Add `docker compose ls` → list compose projects
- [ ] Add `docker compose ps` → show services per project
- [ ] Add `docker compose up/down/stop` actions
- [ ] Wire into DockerPanel as "Compose" tab

**Verification**:
```bash
npm run build
# Compose tab → list projects → start/stop services
```

**Exit criteria**: Docker Compose project management from HWC UI.

---

### Step 4.1: Research Cards
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 1 hour

**Context**: features-list shows "research card rendering" as a chat capability. Messages with link previews or structured data should render as cards.

**Tasks**:
- [ ] Detect URLs and structured data in message content
- [ ] Render as card component (title, description, image, link)
- [ ] Support: URL previews, code search results, file references

**Verification**:
```bash
npm run build
# Send message with URL → renders as research card
```

**Exit criteria**: Research cards render in chat for URLs and structured data.

---

### Step 4.2: Connection Status Handling
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 45 min

**Context**: Hermes WebUI has connection status handling. If WS drops, should show reconnecting state.

**Tasks**:
- [ ] Wire `ws.ts` `onclose` → show "Reconnecting..." banner
- [ ] Implement exponential backoff reconnect (1s, 2s, 4s, max 30s)
- [ ] Show "Connected" / "Reconnecting" / "Disconnected" status in Waybar or right panel header
- [ ] On reconnect, re-subscribe to layout and session state

**Verification**:
```bash
# Kill HWC backend for 5s, restart
# UI shows reconnecting, then reconnects and resumes
```

**Exit criteria**: WS reconnection works with backoff. UI shows correct connection state.

---

### Step 4.3: Message Search
**Status**: pending
**Dependencies**: Step 2.4
**Estimated**: 45 min

**Context**: features-list has "message search" as a chat capability. Need search within session messages.

**Tasks**:
- [ ] Add search icon in ChatPanel header
- [ ] Click → search input overlay
- [ ] Highlight matching messages, navigate with ↑↓
- [ ] Wire to sessionStore search (grep messages for query)

**Verification**:
```bash
npm run build
# ChatPanel → search icon → type query
# Matching messages highlighted, can navigate
```

**Exit criteria**: Message search within session works.

---

### Step 4.4: Session Project Grouping + Color Coding
**Status**: pending
**Dependencies**: Step 2.4
**Estimated**: 1 hour

**Context**: Hermes WebUI has projects (name, color, id). Sessions can be grouped into projects.

**Tasks**:
- [ ] Add projects.json to session store
- [ ] Project model: `{id, name, color, created_at}`
- [ ] SessionsPanel: group sessions by project, color-coded
- [ ] Create/rename/delete projects from SessionsPanel
- [ ] Assign session to project (drag-drop or right-click)

**Verification**:
```bash
npm run build
# SessionsPanel → project groups visible, color coded
# Create project → assign sessions → sessions grouped
```

**Exit criteria**: Projects with color coding. Sessions can be grouped.

---

### Step 5.1: Xpra Manager Implementation
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 2 hours

**Context**: `backend/xpra/manager.go` is a stub. Xpra provides native GUI escape (full desktop apps in HWC tiles). Need real implementation.

**Tasks**:
- [ ] Read `docs/XPRA-INTEGRATION.md` for full architecture
- [ ] Implement `xpra/manager.go`:
  - `Start(display int)` → launch Xvfb + xpra server
  - `Attach(tileID, displayNum)` → xpra attach via unix socket
  - `Detach(tileID)` → xpra detach
  - `Stop()` → shutdown xpra server
- [ ] Use unix socket (`/tmp/.xpra`) for communication
- [ ] Handle SSH tunnel for remote access

**Verification**:
```bash
xpra version  # xpra installed on host
# Start xpra session from HWC → GUI app appears in tile
```

**Exit criteria**: Xpra server starts, attaches, detaches. GUI apps render in HWC tiles.

---

### Step 5.2: XpraTile.svelte
**Status**: pending
**Dependencies**: Step 5.1
**Estimated**: 1 hour

**Context**: XpraTile.svelte exists but is stub. Needs HTML5 client iframe wiring.

**Tasks**:
- [ ] Read XPRA-INTEGRATION.md for frontend spec
- [ ] Wire XpraTile to xpra HTML5 client (`http://localhost:{port}/apps.html`)
- [ ] Handle resize (xpra `--resize=widthxheight` on attach)
- [ ] Keyboard input capture (grab keys for foreign windows)

**Verification**:
```bash
npm run build
# Launch Xpra tile → native GUI app renders inside HWC tile
```

**Exit criteria**: XpraTile shows running GUI app. Resize works.

---

### Step 5.3: SSH Tunnel for Remote Xpra
**Status**: pending
**Dependencies**: Step 5.1
**Estimated**: 1 hour

**Context**: For remote access, xpra traffic should tunnel over SSH.

**Tasks**:
- [ ] Add SSH tunnel creation in xpra/manager.go (`ssh -L 10000:localhost:10000`)
- [ ] XpraTile connects to tunneled port
- [ ] Handle tunnel setup/teardown lifecycle

**Verification**:
```bash
# Start HWC on remote machine → xpra GUI visible locally
```

**Exit criteria**: SSH tunnel for xpra works.

---

### Step 6.1: Real Trace View (Structured Logging)
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 1.5 hours

**Context**: Observability panel has event feed but uses placeholder data. Telemetry subsystem writes JSONL events — need to display structured traces.

**Tasks**:
- [ ] Query telemetry ring buffer for trace events
- [ ] Display trace timeline (timestamp → event type → details)
- [ ] Filter by event type, session ID
- [ ] Expand event to see full payload

**Verification**:
```bash
npm run build
# Observability tab → trace view → real events from telemetry
```

**Exit criteria**: Real trace data visible in observability panel.

---

### Step 6.2: Cost Ledger (Per-Session LLM Cost)
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 1 hour

**Context**: features-list has "cost ledger" as a dashboard capability. Need per-session cost tracking.

**Tasks**:
- [ ] Model: track input/output tokens per message, cost per model
- [ ] Store cost data in session metadata
- [ ] Analytics tab → cost ledger view (daily cost, by model, by session)
- [ ] Use model pricing from config (or hardcoded defaults for common models)

**Verification**:
```bash
npm run build
# Analytics → cost ledger shows per-session LLM costs
```

**Exit criteria**: Cost ledger visible. Real cost data per session.

---

### Step 6.3: Skills Usage Analytics
**Status**: pending
**Dependencies**: Step 0.1
**Estimated**: 45 min

**Context**: features-list has "skills usage tracking". Need to track which skills are invoked.

**Tasks**:
- [ ] Emit telemetry event when skill is loaded/invoked
- [ ] Analytics tab → skills usage chart (top skills, frequency)
- [ ] Per-session skill usage breakdown

**Verification**:
```bash
npm run build
# Analytics → skills tab shows skill usage data
```

**Exit criteria**: Skills usage analytics visible in dashboard.

---

### Step 7.1: OIDC Auth (Keycloak)
**Status**: pending
**Dependencies**: Step 5.1
**Estimated**: 2 hours

**Context**: MULTI-USER-PLAN.md specifies OIDC for multi-user support.

**Tasks**:
- [ ] Read MULTI-USER-PLAN.md for full spec
- [ ] Add OIDC middleware to Go backend (token validation)
- [ ] Add login endpoint → redirect to Keycloak
- [ ] Session isolation: validate user ID from token on every request
- [ ] Middleware for WS connections

**Verification**:
```bash
# Open HWC without token → redirect to login
# With valid token → access granted
```

**Exit criteria**: OIDC login flow works. Unauthorized users redirected to login.

---

### Step 7.2: Per-User Session Store Isolation
**Status**: pending
**Dependencies**: Step 7.1
**Estimated**: 1 hour

**Context**: Each user should only see their own sessions and workspaces.

**Tasks**:
- [ ] Prefix session paths with `sessions/{userID}/`
- [ ] Middleware extracts userID from OIDC token
- [ ] All session operations scoped to user directory

**Verification**:
```bash
# Two users → each sees only their own sessions
```

**Exit criteria**: User isolation enforced. Users can't access each other's sessions.

---

### Step 7.3: Coder Workspace Lifecycle
**Status**: pending
**Dependencies**: Step 7.1
**Estimated**: 2 hours

**Context**: MULTI-USER-PLAN.md specifies Coder workspace provisioning.

**Tasks**:
- [ ] Workspace CRD: create workspace container, assign to user
- [ ] Suspend: pause container, store state
- [ ] Delete: remove container, cleanup volumes
- [ ] Status: running/stopped/suspended per workspace

**Verification**:
```bash
# Admin → create workspace → provisioned for user
# Suspend → resume → state preserved
# Delete → workspace removed
```

**Exit criteria**: Coder workspace CRUD from admin panel.

---

## Verification Commands (All Phases)

```bash
# Build
cd /home/hermeswebui/.hermes/hermes-web-computer
go build ./... && npm run build

# Tests
go test ./backend/... -count=1 -timeout=120s
playwright test --project=chromium 2>&1 | tail -30

# Manual smoke test
# 1. Frontend loads at http://localhost:3005
# 2. Chat sends message → receives streamed reply
# 3. DockerPanel lists real containers
# 4. Workspace file tree shows files
# 5. Slash command autocomplete works
```

---

## Anti-Patterns to Avoid

- **Don't touch agent-os unless migrating features**: agent-os is the source to migrate FROM, not maintain alongside
- **Don't break existing tiles**: Terminal, Monaco, Browser, Chat all work. Test each phase with all tiles.
- **Don't make Docker blocking**: docker operations should be async (non-blocking WS methods)
- **Don't skip the WS connection fix**: everything depends on Step 0.1
- **Don't add dependencies**: keep npm packages minimal. No new frameworks.

---

## Files to Create/Modify

| Step | Create | Modify |
|------|--------|--------|
| 0.1 | — | `frontend/src/stores/ws.ts`, `frontend/vite.config.ts` |
| 0.2 | — | `backend/cmd/server/main.go` (or entry point) |
| 1.1 | `frontend/src/components/DockerPanel.svelte` | `RightPanel.svelte` (add tab) |
| 1.2 | — | `backend/ws/multiplexer.go` (paths) |
| 1.3 | — | `backend/security/security.go` |
| 2.1 | `frontend/src/components/FileTree.svelte` (wire) | existing FileTree + PreviewPanel |
| 2.2 | `frontend/src/stores/commands.svelte.ts` | `ChatPanel.svelte` (add autocomplete) |
| 2.3 | `frontend/src/components/FileUpload.svelte` | `ChatPanel.svelte` |
| 2.4 | — | `SessionsPanel.svelte` (add search/rename/delete) |
| 2.5 | — | `ChatPanel.svelte` (add context meter) |
| 3.1 | — | `DockerPanel.svelte` (add create dialog) |
| 3.3 | — | `DockerPanel.svelte` (add images tab) |
| 3.4 | — | `DockerPanel.svelte` (add compose tab) |
| 5.1 | `backend/xpra/manager.go` (implement) | — |
| 5.2 | — | `frontend/src/components/XpraTile.svelte` |
| 7.1 | `backend/auth/oidc.go` (new) | `backend/ws/multiplexer.go` (middleware) |

---

## Cron Jobs to Fix

After Phase 0, update broken cron jobs:

```bash
# Current wrong paths:
# rebuild+deploy → /opt/data/hermes-web-computer (wrong)
# canary watch → :3001 (wrong)
# visual QA → :3113 (wrong)

# Correct paths:
# rebuild+deploy → /home/hermeswebui/.hermes/hermes-web-computer
# canary watch → :3005
# visual QA → :3005 (or disable if redundant)
```

See `cronjob(action='list')` for IDs, then `cronjob(action='update', job_id=..., prompt=...)` to fix.

---

## Notes

- **Port 3005**: HWC Go backend (target for frontend WS)
- **Port 8642**: Hermes Agent (upstream LLM router)
- **Port 5432**: PostgreSQL (agent-os DB — not used by HWC)
- **Session data**: `~/.hermes/hermes-web-computer/sessions/` (target after migration)
- **Hermes WebUI**: replace, don't extend. HWC is the new default UI.
- **Testing**: Playwright E2E + `go test` after each phase. No regressions.