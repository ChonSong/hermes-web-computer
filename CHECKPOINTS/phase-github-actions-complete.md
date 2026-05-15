# Phase 5 Complete: GitHub Actions CI Workflow

**Date:** 2026-05-15  
**Commit:** `ci(phase5): add GitHub Actions CI workflow`  
**Branch:** `main` (push pending)

---

## What was done

Replaced the existing minimal `.github/workflows/ci.yml` (23 lines, Go vet + build only) with a comprehensive multi-phase CI pipeline covering lint, build, test, E2E, a11y, visual regression, and nightly audits.

### New CI Pipeline Structure

#### Phase 1: `lint-and-build` (matrix job)
- **Go backend**: `go mod download` → `go vet ./...` → `go build -o /tmp/agent-os ./cmd/server` → `go test -short ./...`
- **Frontend**: `npm ci` → `npm run build`
- **E2E deps**: `npm ci` at root `e2e/` directory
- **Bundle size check**: validates largest JS chunk < 500KB threshold
- **Matrix**: Go 1.23/1.24 × Node.js 20/22 (4 combinations, no fail-fast)

#### Phase 2: `e2e` (Playwright, 2 shards)
- Starts Go backend on port 3005 (with health-check wait loop)
- Starts Vite dev server on port 5173 (with health-check wait loop)
- Runs `cd e2e && npx playwright test --shard=N/2` (tests split across 2 parallel shards)
- Uploads `e2e/test-results/` as artifacts (7-day retention)
- Runs only after `lint-and-build` completes (via `needs:`)

#### Phase 3: `a11y` (Accessibility)
- Starts Vite dev server on port 5173
- `npx playwright test tests/a11y` (keyboard nav, screen reader, contrast tests)
- Uploads artifacts
- Runs only after `lint-and-build` completes

#### Phase 4: `visual` (Visual Regression)
- Starts Vite dev server on port 5173
- `npx playwright test tests/visual` (baseline + regression)
- Uploads artifacts
- Runs only after `lint-and-build` completes

#### Phase 5: `nightly` (Scheduled)
- Triggered via `github.event_name == 'schedule'` at 03:00 UTC daily
- Runs `treosh/lighthouse-ci-action@v11` with `lighthouserc.json`
- Uploads Lighthouse HTML reports as artifacts

### Additional improvements
- **Concurrency group** `ci-${{ github.ref }}` — auto-cancels in-progress runs on new pushes to the same branch/PR
- **cancel-in-progress: true** — prevents wasted CI minutes on stale branches
- **Artifact retention**: 7 days for all test result uploads
- **fail-fast: false** — all matrix/parallel jobs continue even if one fails (for maximum visibility)

---

## Files created/modified

| File | Change |
|------|--------|
| `.github/workflows/ci.yml` | **Modified** (596 bytes → 8243 bytes) |

---

## Commit message

```
ci(phase5): add GitHub Actions CI workflow

Comprehensive 5-phase CI pipeline:
- Phase 1 (lint-and-build): Go vet/build/test × Node 20/22 matrix, bundle size check
- Phase 2 (e2e): Playwright tests in 2 shards, backend + Vite server lifecycle
- Phase 3 (a11y): keyboard/screen-reader/contrast Playwright tests
- Phase 4 (visual): baseline + regression visual tests
- Phase 5 (nightly): Lighthouse audit via lighthouse-ci-action

Concurrency auto-cancel on duplicate pushes. Artifact retention 7 days.
```

---

## Push status

**Not yet pushed.** Requires `git push` after commit.

---

## Next action

Execute Phase 6 (pre-commit hooks): Add `.husky/pre-commit` with `go vet` + `go test` + `vite build` + `playwright smoke test`.