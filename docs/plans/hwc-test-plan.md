# HWC Playwright E2E Test Plan

## Current State Assessment

### Existing Test Suite (e2e/tests/)
| Category | Files | Tests | Status |
|----------|-------|-------|--------|
| **Smoke** | smoke.spec.ts | 8 tests (a-h) | ✅ All passing (Phase 14 verified) |
| **Layout** | 01-layout.spec.ts | 1 test | ✅ Passing |
| **Resize** | 02-resize.spec.ts | 1 test | ✅ Passing |
| **Workflows** | hwc-features.spec.ts | 13 tests (DockerPanel, slash commands, Ctrl+K) | ✅ Passing |
| **Workflows** | cross-panel.spec.ts | 4 tests (terminal→file tree→editor) | ✅ Passing |
| **Workflows** | file-edit.spec.ts | 2 tests (Monaco edit + save) | ⚠️ Fragile (deep path nav) |
| **Workflows** | chat-context.spec.ts | 6 tests (message send, ordering, empty) | ✅ Passing |
| **Workflows** | recovery.spec.ts | 4 tests (reconnect, navigation) | ⚠️ Weak assertions |
| **Workflows** | pipeline.spec.ts | 4 tests (multi-terminal, dock split) | ⚠️ Conditional skips |
| **A11y** | contrast.spec.ts | 3 tests (WCAG AA contrast) | ⚠️ Pre-existing failures |
| **A11y** | keyboard.spec.ts | (exists) | Not assessed |
| **A11y** | screen-reader.spec.ts | (exists) | Not assessed |
| **Chaos** | concurrent.spec.ts | (exists) | ❌ Pre-existing failures |
| **Chaos** | network.spec.ts | (exists) | Not assessed |
| **Chaos** | ws-flood.spec.ts | (exists) | ❌ Pre-existing (CONNECTING race) |
| **Chaos** | server-death.spec.ts | (exists) | Not assessed |
| **Perf** | load-time.spec.ts | (exists) | Not assessed |
| **Visual** | baseline.spec.ts | (exists) | Not assessed |
| **Visual** | regression.spec.ts | (exists) | Not assessed |
| **TOTAL** | **18 files** | **~50+ tests** | **24/24 core passing** |

### Known Pre-Existing Failures (from Phase 14)
1. `a11y/contrast.spec.ts:58` — Playwright rejects `page.evaluate()` multi-arg (API compat)
2. `chaos/concurrent.spec.ts` — Uses outdated `.bg-gray-950` CSS selector
3. `chaos/ws-flood.spec.ts:95` — WebSocket CONNECTING state race condition

### Chronically Failing Cron Jobs (now fixed)
1. **canary watch** — Was using `browser_navigate` from container → can't reach host:3005 → **FIXED** (switched to curl)
2. **rebuild+deploy** — Was using `workdir: /opt/data/...` (doesn't exist in container) + `go build` (Go not installed) → **FIXED** (corrected workdir, Go awareness, added E2E step)
3. **nightly build health** — Was SSH with wrong key path + wrong port → **FIXED** (fallback chain, correct key, container-first approach)

---

## Test Plan — v1.4 Coverage Matrix

### P0 — Critical Path (must pass for every commit)
| Test ID | Scenario | Area | File | Priority |
|---------|----------|------|------|----------|
| P0-1 | App loads, #app visible, correct URL | Smoke | smoke.spec.ts | P0 |
| P0-2 | WebSocket connects, "Connected" state shown | Smoke | smoke.spec.ts | P0 |
| P0-3 | Dark theme applied (bg-gray-950 or darker) | Smoke | smoke.spec.ts | P0 |
| P0-4 | Three-column layout renders (left, middle, right panels) | Layout | 01-layout.spec.ts | P0 |
| P0-5 | Terminal tile visible xterm.js loaded | Smoke | smoke.spec.ts | P0 |
| P0-6 | Chat input visible, can type and send message | Chat | chat-context.spec.ts | P0 |
| P0-7 | Ctrl+K opens command palette | Keyboard | hwc-features.spec.ts | P0 |
| P0-8 | Ctrl+B toggles left panel | Keyboard | smoke.spec.ts | P0 |
| P0-9 | Ctrl+? opens keymap overlay | Keyboard | smoke.spec.ts | P0 |
| P0-10 | File tree in LeftPanel navigates directories | Files | cross-panel.spec.ts | P0 |
| P0-11 | Monaco editor opens file, shows content | Editor | file-edit.spec.ts | P0 |
| P0-12 | DockerPanel tab visible, shows Containers/Images/Compose | Docker | hwc-features.spec.ts | P0 |
| P0-13 | Disconnect → reconnect recovery | Resilience | recovery.spec.ts | P0 |
| P0-14 | Page reload preserves layout | Recovery | recovery.spec.ts | P0 |

### P1 — Core Features (should pass, failures block release)
| Test ID | Scenario | Area | File | Priority |
|---------|----------|------|------|----------|
| P1-1 | Panel resize (drag handle changes width) | Layout | 02-resize.spec.ts | P1 |
| P1-2 | Command palette filters by typing | Keyboard | hwc-features.spec.ts | P1 |
| P1-3 | Command palette arrow key nav + Enter select | Keyboard | hwc-features.spec.ts | P1 |
| P1-4 | Escape closes command palette | Keyboard | hwc-features.spec.ts | P1 |
| P1-5 | Slash command mode in chat input | Chat | hwc-features.spec.ts | P1 |
| P1-6 | Multiple chat messages all appear in order | Chat | chat-context.spec.ts | P1 |
| P1-7 | Empty/whitespace message not sent | Chat | chat-context.spec.ts | P1 |
| P1-8 | Create file in terminal → appears in file tree | Cross-panel | cross-panel.spec.ts | P1 |
| P1-9 | Create dir + file in terminal → navigate in tree → open in editor | Cross-panel | cross-panel.spec.ts | P1 |
| P1-10 | Terminal output visible (echo command response) | Terminal | pipeline.spec.ts | P1 |
| P1-11 | Launch app from dock → tile count increases | Layout | pipeline.spec.ts | P1 |
| P1-12 | Close tile (Shift+Q) → tile reflow | Layout | pipeline.spec.ts | P1 |
| P1-13 | Kill server → frontend shows "Disconnected" | Chaos | server-death.spec.ts | P1 |
| P1-14 | Restart server → frontend auto-reconnects | Chaos | network.spec.ts | P1 |

### P2 — Edge Cases & Quality (fix in next sprint)
| Test ID | Scenario | Area | File | Priority |
|---------|----------|------|------|----------|
| P2-1 | WCAG AA contrast ≥4.5:1 on all text elements | A11y | contrast.spec.ts | P2 |
| P2-2 | Focus indicator visible on Tab navigation | A11y | contrast.spec.ts | P2 |
| P2-3 | Keyboard-only navigation to all panels | A11y | keyboard.spec.ts | P2 |
| P2-4 | Screen reader landmarks present | A11y | screen-reader.spec.ts | P2 |
| P2-5 | Concurrent WS ops don't corrupt state | Chaos | concurrent.spec.ts | P2 |
| P2-6 | WS flood (100 msg/sec) doesn't crash | Chaos | ws-flood.spec.ts | P2 |
| P2-7 | Network disconnect → throttle → reconnect | Chaos | network.spec.ts | P2 |
| P2-8 | Load time <3s on localhost | Perf | load-time.spec.ts | P2 |
| P2-9 | Visual regression (screenshot diff) | Visual | regression.spec.ts | P2 |
| P2-10 | Multi-line file edit + save + re-open persistence | Editor | file-edit.spec.ts | P2 |
| P2-11 | Workspace pill shows 9 workspaces | Layout | NEW v1.4 | P2 |

### v1.4 New Features (not yet tested)
| Test ID | Scenario | Area | File | Priority |
|---------|----------|------|------|----------|
| N-1 | Workspace switcher (pill dots 1-9) | Workspace | NEW smoke | P0 |
| N-2 | Session panel (Sessions tab in RightPanel) | Session | NEW smoke | P1 |
| N-3 | Profile panel (Profiles tab in RightPanel) | Profile | NEW smoke | P1 |
| N-4 | Skills panel (Skills tab in RightPanel) | Skills | NEW smoke | P1 |
| N-5 | Cron panel (Crons tab in RightPanel) | Cron | NEW smoke | P1 |
| N-6 | Memory panel (Memory tab in RightPanel) | Memory | NEW smoke | P1 |
| N-7 | Config panel (Settings tab + model picker) | Config | NEW smoke | P1 |
| N-8 | Dark theme toggle (7 themes) | Theme | NEW smoke | P1 |
| N-9 | Observability panel (event feed) | Observability | NEW smoke | P1 |
| N-10 | Dock app icons (Calculator, Browser, Editor, etc.) | Dock | NEW smoke | P0 |
| N-11 | App launcher panel (Apps tab in LeftPanel) | Apps | NEW smoke | P1 |

---

## Test Infrastructure Requirements

### Playwright Config Updates
```typescript
// e2e/playwright.config.ts — needs updates for:
1. webServer.command must use HERMES_HWC_ROOT env var
2. Add projects: chromium (headless for CI), chromium-visible (headed for debug)
3. Add global setup for test fixtures (create test files, clean state)
4. Increase timeout for slow terminal operations (15s → 30s for xterm.js interactions)
5. Add retry: 1 for flaky terminal tests
6. Viewport: 1440x900 (matches AGENTS.md screenshot spec)
```

### Test Fixtures Needed
```typescript
// e2e/fixtures/hwc-fixture.ts
- cleanTerminalState() — reset PTY before each test
- createTestFile(path, content) — setup helper
- waitForConnected(page) — shared WS connection wait
- focusTerminal(page) — click terminal tile and wait for xterm
```

### CI/CD Integration
```yaml
# Run pipeline:
1. npx playwright test e2e/tests/smoke.spec.ts --reporter=list  (P0, <2min)
2. npx playwright test e2e/tests/01-layout.spec.ts e2e/tests/02-resize.spec.ts --reporter=list (P0, <3min)
3. npx playwright test e2e/tests/workflows/ --reporter=list (P1, <5min)
4. npx playwright test e2e/tests/chaos/ --reporter=list (P2, <3min)
5. npx playwright test e2e/tests/a11y/ --reporter=list (P2, <2min)

# Cron integration:
- Rebuild job now runs E2E after verification
- Nightly build health runs E2E as part of health check
- Canary job checks server + reports status
```

### Screenshot/Visual QA
```
e2e/scripts/visual-qa.sh:
- google-chrome-stable --headless --screenshot=/tmp/hwc-qa/ http://localhost:3005
- Compare against e2e/test-results/visual/baseline.png
- Report diff percentage
```

---

## Execution Plan

### Phase 1 (This Session) — Fix cron jobs ✅
- [x] Fix canary watch job (curl instead of browser_navigate)
- [x] Fix rebuild+deploy job (correct workdir, Go awareness, E2E step)
- [x] Fix nightly build health job (SSH fallback chain, correct paths)

### Phase 2 (This Session) — Stabilize existing tests
- [ ] Fix `contrast.spec.ts:58` — page.evaluate() multi-arg issue
- [ ] Fix `concurrent.spec.ts` — update `.bg-gray-950` selector
- [ ] Fix `ws-flood.spec.ts:95` — CONNECTING state race
- [ ] Strengthen `recovery.spec.ts` — add real WS close test
- [ ] Strengthen `file-edit.spec.ts` — more reliable file tree navigation
- [ ] Strengthen `pipeline.spec.ts` — remove conditional skips

### Phase 3 — v1.4 new feature tests
- [ ] Write smoke tests for new RightPanel tabs (Sessions, Profiles, Skills, Cron, Memory, Config)
- [ ] Write smoke tests for Workspace switcher
- [ ] Write smoke tests for Dock app icons
- [ ] Write smoke tests for Theme switcher
- [ ] Write smoke tests for App launcher panel
- [ ] Write smoke tests for Observability panel

### Phase 4 — Infrastructure hardening
- [ ] Update playwright.config.ts with proper fixtures, timeouts, projects
- [ ] Add global setup/teardown for test isolation
- [ ] Add visual QA script (baseline screenshot comparison)
- [ ] Add E2E results parsing to cron job reports
- [ ] Add flaky test retry logic
- [ ] Document test writing conventions in AGENTS.md

---

## Success Criteria

| Metric | Current | Target |
|--------|---------|--------|
| Core tests passing | 24/24 (100%) | 24/24 (100%) — must maintain |
| All P0+P1 tests | ~35 tests, ~30 passing | 35/35 (100%) |
| P2 tests (edge cases) | ~15 tests, ~10 passing | 15/15 (100%) |
| v1.4 new features tested | 0/11 | 11/11 (100%) |
| Cron job reliability | 0/3 passing | 3/3 passing (canary, rebuild, nightly) |
| Visual QA coverage | None | Screenshot baseline + regression |
| Total test count | ~50 | ~75+ |
