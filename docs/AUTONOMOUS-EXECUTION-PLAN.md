# Hermes Web Computer — Autonomous Execution Plan

> This plan drives 4 cron jobs that complete hermes-web-computer v1.0 tiles unattended.
> Each phase writes a completion marker: `echo "PHASE_N_COMPLETE" > /tmp/hermes-computer-phase-N.done`

## Phase 1: Complete 90%-done features (Editor save, Agent chat, Voice UI)

### Phase 1 Checklist
- [ ] `Monaco.svelte` — add save/write-back via `fs.write` (Ctrl+S / Cmd+S keyboard shortcut)
- [ ] `RightPanel.svelte` — add voice toggle (mic record/stop buttons)
- [ ] `ws.ts` store — add `audio.start()` and `audio.stop()` WS methods
- [ ] Backend: wire `chat.send` to real Hermes API (HTTP POST to host.docker.internal:8642 or Hermes webhook)
- [ ] Backend: wire `audio.stream` → Fun-Audio-Chat bridge (already exists in audio/bridge.go, just needs frontend mic capture)
- [ ] Test: `make test` or `go test ./...` passes
- [ ] Test: `npx playwright test e2e/tests/01-layout.spec.ts` passes
- [ ] Commit + push all changes

### Phase 1 Completion Criteria
1. Monaco editor can save files via Ctrl+S → writes to disk via backend
2. Right panel has record/stop mic buttons
3. Agent chat sends to real Hermes backend (not echo)
4. All existing tests pass
5. `echo "PHASE_1_COMPLETE" > /tmp/hermes-computer-phase-1.done`

---

## Phase 2: Browser tile (chromedp integration)

### Phase 2 Checklist
- [ ] Create `backend/browser/browser.go` — chromedp wrapper with navigate/screenshot/interact
- [ ] Add WS protocol methods: `browser.navigate`, `browser.screenshot`, `browser.click`, `browser.input`
- [ ] Create `frontend/src/components/Browser.svelte` — iframe-like tile with URL bar, back/forward, navigation controls
- [ ] Update `Tile.svelte` to handle `content === 'browser'`
- [ ] Update `apps.go` to add `{ID: "browser", Name: "Browser"}` to apps list
- [ ] Update `apps.launch` handler for browser type (spawn new chromedp context)
- [ ] Test: `go test ./...` passes (or at least compiles cleanly)
- [ ] Commit + push

### Phase 2 Completion Criteria
1. Browser tile renders in middle column when launched
2. URL bar accepts input, navigates via chromedp
3. Screenshots relay from Go backend to frontend via WS
4. `echo "PHASE_2_COMPLETE" > /tmp/hermes-computer-phase-2.done`

---

## Phase 3: Dashboard migration via repo-transmute

### Phase 3 Checklist
- [ ] Run `repo-transmute v2 ingest /home/sean/.hermes/agent-os` (or clone ChonSong/agent-os)
- [ ] Extract 4-6 key React components as migration targets
- [ ] Run `repo-transmute v2 migrate` to convert them to Svelte 5 runes
- [ ] Integrate migrated components as dashboard tiles in hermes-web-computer frontend
- [ ] Wire up any API calls to Go backend
- [ ] Vision verification: screenshot migrated tiles, compare to source
- [ ] Commit + push

### Phase 3 Completion Criteria
1. 4+ React components from agent-os successfully migrated to Svelte 5
2. Migrated components render as tiles in hermes-web-computer
3. Vision score > 80% similarity to source
4. `echo "PHASE_3_COMPLETE" > /tmp/hermes-computer-phase-3.done`

---

## Phase 4: Polish and test coverage

### Phase 4 Checklist
- [ ] Implement `tool.execute` handler in backend
- [ ] Implement `fs.watch` (optional, if time permits)
- [ ] Add E2E tests for Browser tile
- [ ] Add E2E tests for Voice recording/playback
- [ ] Run full `npx playwright test` suite
- [ ] Update PLAN.md and APPLICATION-PLAN.md with current status
- [ ] Commit + push

### Phase 4 Completion Criteria
1. All v1.0 tiles (Terminal, Browser, Voice, Dashboard) functional
2. E2E test suite passes for all tiles
3. `echo "PHASE_4_COMPLETE" > /tmp/hermes-computer-phase-4.done`

---

## Execution Notes
- Work directory: `/opt/data/hermes-web-computer`
- Go backend: `backend/` — run `go build ./...` to verify compilation
- Frontend: `frontend/` — run `npm run build` to verify Svelte compilation
- Tests: `npx playwright test` from repo root
- Git: commit frequently with descriptive messages, push to `main`
- If a phase fails: write error details to `/tmp/hermes-computer-phase-N.error` and exit 0 (don't block next phase)
- Use `hermes-computer-planning/APPLICATION-PLAN.md` for architectural decisions (stack, tile specs, etc.)
- Use `hermes-computer-planning/ONE-WEBSITE.md` for the overarching vision
- Use `repo-transmute v2` (at `/opt/data/repo-transmute-v2` or install via pip) for Phase 3 migration
