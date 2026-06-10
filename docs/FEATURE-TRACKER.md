# FEATURE-TRACKER.md — hermes-web-computer

> Single source of truth for feature status. Update after every commit.

**Last updated:** 2026-06-09
**Current HEAD:** `8328ad2` (feat(phase6): cost ledger per session + skills usage analytics)
**Server:** ✅ Running on host port 3005 (HTTP 200)
**Tunnel:** ⚠️ No Cloudflare tunnel for HWC yet (needs manual setup)
**Go tests:** ✅ All passing (6 packages: audio, layout, security, session, ws, agent)

---

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Complete and merged |
| 🟡 | In progress (actively being worked) |
| ⚪ | Not started |
| ❌ | Blocked / broken |
| 🔶 | Partially complete (known gaps) |

---

## Waybar + Shell (Priority 1)

Based on `docs/WAYBAR-SPEC.md` — Hyprland reference screenshot functional spec.

| Feature | Spec section | Status | Last Updated | Notes |
|---------|-------------|--------|-------------|-------|
| Waybar top bar | WAYBAR-SPEC.md §2 | ✅ Complete | 2026-05-24 | Waybar.svelte replaces WorkspacePill — 9 clickable dots, window title, agent status, clock |
| Clickable workspace indicators | WAYBAR-SPEC.md §4 | ✅ Complete | 2026-05-24 | Click dot → setActiveWorkspace(n) — keyboard still works (Shift+1-9) |
| Window title in Waybar | WAYBAR-SPEC.md §2 | ✅ Complete | 2026-05-24 | Subscribes to ui.focus.changed WS event |
| System tray (wifi/volume/battery/temp) | WAYBAR-SPEC.md §2.3 | ✅ Complete | 2026-05-24 | Backend extended: Wifi/Battery/Volume fields in SystemMetrics; frontend wired: dim on disconnect, red on critical |
| Dock: click-to-launch tiles | WAYBAR-SPEC.md §3 | ✅ Complete | 2026-05-24 | Click tile items → apps.launch + layout.update split; panel items dispatch hwc-dock-panel event |
| Dock: running indicator dot | WAYBAR-SPEC.md §3.2 | ✅ Complete | 2026-05-24 | Purple dot for running tiles; white/40 dot for pinned-but-not-running; layout tree subscription tracks active tiles |
| Dock: pin/unpin apps | WAYBAR-SPEC.md §3.3 | ✅ Complete | 2026-05-24 | Right-click context menu with pin/unpin toggle, new instance, focus/launch options |
| File explorer sidebar | WAYBAR-SPEC.md §5 | ✅ Complete | 2026-05-24 | VSCode-style collapsible tree with ▶/▼ chevrons; `expandedPaths` Set tracks open dirs; `currentPath` with breadcrumb nav |
| File explorer context menu | WAYBAR-SPEC.md §5.5 | ✅ Complete | 2026-05-24 | Right-click: Open/Rename/Delete/Copy Path; inline rename with Enter/Escape; delete confirmation |
| Bottom terminal panel tabs | WAYBAR-SPEC.md §6 | ✅ Complete | 2026-05-24 | BottomPanel.svelte: Terminal/Problems/Output/Ports tabs, drag-to-resize, Ctrl+` toggle, multi-tab support |
| Bottom panel resize | WAYBAR-SPEC.md §6.1 | ✅ Complete | 2026-05-24 | Drag handle at top, 120–600px range, persists to localStorage |
| Menu bar | WAYBAR-SPEC.md §9 | ✅ Complete | 2026-05-24 | MenuBar.svelte — File/Edit/View/Go/Run/Terminal/Help with keyboard shortcuts; Alt+F/E/V/G/R/T/H to open menus |

**Backend dependencies (for system tray):**
| Dependency | Status | Notes |
|------------|--------|-------|
| Host metrics endpoint (CPU/mem/net/temp) | ✅ Complete | `/api/system/metrics` + WS `system.metrics` returning host data |
| Audio status from Fun-Audio-Chat | 🔶 partial | Subscribed to audio state events; real volume data via PipeWire/PulseAudio/ALSA fallback chain |

---

## App Tiles

| Tile | Status | Last Updated | Notes |
|------|--------|-------------|-------|
| Terminal (xterm.js) | ✅ Complete | v1.2 | Lazy-loaded, PTY via backend |
| Monaco Editor | ✅ Complete | v1.2 | Read-only with Ctrl+S write-back |
| Browser (chromedp) | ✅ Complete | v1.2 | Navigate/click/input/screenshot |
| Chat Panel | ✅ Complete | v1.2 | Hermes Agent SSE streaming |
| File Manager (Dash) | ✅ Complete | v1.2 | Browse/preview/edit/create/delete |
| System Status (Dash) | ✅ Complete | v1.3 | Real sysinfo from Phase 9 backend handlers |
| Analytics (Dash) | ✅ Complete | v1.4 | Phase 9 telemetry + Phase 6.2 cost ledger + 6.3 skills analytics (Tok/Model/Skills tabs) |
| Observability (Dash) | ✅ Complete | v1.3 | Real event feed from Phase 9 dashboard.stats |
| File Upload | ✅ Complete | 2026-06-10 | Standalone dock tile with drag-drop, browse, file grid, progress; also integrated into ChatPanel |
| Calculator | ⚪ not-started | — | Simple web app tile |
| Calendar | ⚪ not-started | — | Web app tile |
| Music Player | ⚪ not-started | — | Spotify integration possible |
| Camera | ⚪ not-started | — | MediaDevices API |

---

## Backend Packages

| Package | File | Status | Notes |
|---------|------|--------|-------|
| WebSocket Multiplexer | `ws/multiplexer.go` | ✅ Complete | ui/agent/audio protocols |
| PTY Supervisor | `pty/supervisor.go` | ✅ Complete | 1MB ring buffer + checkpoint |
| Layout Tree | `layout/tree.go` | ✅ Complete | Split/mount/unmount/resize/swap/fullscreen |
| Security Enforcer | `security/security.go` | ✅ Complete | YAML tiers + token gating |
| Session Store | `session/store.go` | ✅ Complete | JSON file-based + cost tracking (Phase 6.2) |
| Browser Manager | `browser/browser.go` | ✅ Complete | chromedp headless, wired to apps.launch + 7 WS route handlers |
| LLM Router | `llm/router.go` | ✅ Complete | Multi-provider (not yet wired to UI) |
| MCP Client | `mcp/client.go` | ✅ Complete | JSON-RPC 2.0 stdio (not yet wired to UI) |
| Audio Bridge | `audio/bridge.go` | ✅ Complete | Fun-Audio-Chat relay |
| Telemetry | `telemetry/telemetry.go` | ✅ Complete | JSONL ring buffer at stateDir/telemetry/, syncer configurable via TELEMETRY_ENDPOINT |
| Config Manager | `config/manager.go` | ✅ Complete | YAML read/write + env vars |
| Docker Manager | `docker/manager.go` | ✅ Complete | CLI wrapper |
| Agent Streamer | `agent/streamer.go` | ✅ Complete | SSE client for Hermes |
| Host Metrics | `ws/metrics.go` | ✅ Complete | CPU/mem/net/temp for Waybar |

---

## xpra Escape Hatch

> ✅ Implemented in Phase 5 — commit `01c1fec`

| Component | Plan doc | Status | Last Updated | Notes |
|-----------|----------|--------|-------------|-------|
| xpra server setup script | XPRA-INTEGRATION.md | ✅ Complete | 2026-06-10 | `scripts/setup-xpra.sh` — installs xpra, creates systemd service, starts on :10 with HTML5 on 9453 |
| `xpra/manager.go` | XPRA-INTEGRATION.md | ✅ Complete (uncommitted) | 2026-05-24 | Go backend package for xpra session lifecycle — StartServer/StopServer/AttachApp/ListWindows/DetachWindow. 342 lines. Code exists on disk but NOT committed. |
| `xpra/proxy.go` | XPRA-INTEGRATION.md | ✅ Complete (uncommitted) | 2026-05-24 | HTTP reverse proxy + WebSocket proxy (httputil.ReverseProxy + nhooyr/websocket bidirectional relay). 170 lines. Code exists on disk but NOT committed. |
| `XpraTile.svelte` | XPRA-INTEGRATION.md | ✅ Complete | 2026-05-24 | Iframe tile component, 413 lines (exists on disk, status icon, reconnect logic, window list bar) |
| Middleware wiring in `ws/multiplexer.go` | XPRA-INTEGRATION.md | ✅ Complete (uncommitted) | 2026-05-24 | `xpra.start`/`xpra.stop`/`xpra.attach`/`xpra.list`/`xpra.detach` WS handlers + proxy routes. Code exists on disk but NOT committed. |

---

## Multi-User / Team Support

| Component | Plan doc | Status | Notes |
|-----------|----------|--------|-------|
| Multi-user architecture | MULTI-USER-PLAN.md | ⚪ not-started | Per-user layout trees |
| OIDC auth (Keycloak) | MULTI-USER-PLAN.md §M1 | ⚪ not-started | Token validation middleware |
| Coder workspace lifecycle | MULTI-USER-PLAN.md §M2 | ⚪ not-started | Provision/suspend/delete |
| Shared tiles (collaborative) | MULTI-USER-PLAN.md §M3 | ⚪ not-started | CRDT or leader-follower |
| RBAC + audit | MULTI-USER-PLAN.md §M4 | ⚪ not-started | Admin/user/viewer roles |

---

## Infrastructure

| Item | Status | Notes |
|------|--------|-------|
| Go tests (45+) | ✅ Passing | `go test ./... -count=1 -timeout=120s` |
| Playwright E2E (17+) | ✅ Passing | Layout, resize, chaos, a11y, perf |
| Theme (solid #191919) | ✅ Complete | ΔE<8 vs reference, verified 2026-05-24 |
| CI pipeline | ✅ Passing | GitHub Actions |
| Cloudflare tunnel | ✅ Running | hermes-webui tunnel |
| Port | ✅ Correct | HWC runs on **3005** (all legacy refs fixed) |

---

## Cron Jobs

| Job | Schedule | Status | Last Run | Notes |
|-----|----------|--------|----------|-------|
| Rebuild + deploy check | every 240m | ✅ OK | 2026-06-08 | Re-enabled after server recovery |
| HWC canary watch | every 360m | ✅ OK | 2026-06-09 | Re-enabled after server recovery |
| HWC Visual QA | every 720m | ❌ Missing | — | Was removed/never re-created after initial setup |
| Phase Engine | hourly | ⚪ Disabled | — | All phases complete; no longer needed |
| Nightly build health | 0 19 * * * | ✅ OK | 2026-06-08 | Running on schedule |

---

## Open Questions

| Question | Status | Resolution |
|----------|--------|------------|
| Waybar glassmorphism or solid? | ✅ Resolved | Solid #191919 theme implemented and verified |
| Workspace 0 as scratchpad? | ⚪ | Not implemented — currently 9 workspaces (1-9) |
| `fs.watch` for file changes? | ⚪ | Marked future in PLAN.md |
| Dashboard tiles real data? | ✅ Complete | Backend handlers wired to system metrics / analytics / stats |

---

## Next Action

**Priority:** Complete v1.4 — address remaining gaps.

**Phase 6 gaps now closed:** Cost ledger (6.2) + Skills analytics (6.3) ✅ delivered.

**Remaining for v1.4:**
1. Context meter / message search (deferred)
2. Cloudflare tunnel for HWC port 3005 (needs manual setup)
3. Fresh E2E test run + visual QA baseline screenshots

**Next major phase:** Phase 7 — Multi-user / OIDC auth (MULTI-USER-PLAN.md)