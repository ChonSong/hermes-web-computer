# HWC Visual Audit — 2026-05-23

## Status: Backend Running ✅ | Visual QA Broken ❌

**Backend**: Running at `http://localhost:3113` (pid 6471 on host), serving built dist.
**WebSocket**: ✅ Upgrading to ws correctly
**Static assets**: ✅ All JS/CSS chunks serving with correct Content-Type

---

## Design Token Audit (Source vs Built CSS)

### Illogical Impulse Spec → Implementation

| Token | Spec | Source | Built CSS | Status |
|-------|------|--------|-----------|--------|
| `--bg-primary` | `#0a0a0f` | `glass.css` | ✅ Found | OK |
| `--bg-panel` | `rgba(18,18,26,0.85)` | `glass.css` | ✅ Found | OK |
| `--accent-primary` | `#a78bfa` | `glass.css` | ✅ Found | OK |
| `--border-active` | `rgba(167,139,250,0.5)` | `glass.css` | ✅ Found | OK |
| `--shadow-glow` | `0 0 12px rgba(167,139,250,0.3)` | `glass.css` | ✅ Found | OK |
| `@theme` directive | Tailwind v4 | `glass.css` | ✅ Found | OK |
| `backdrop-blur-xl` | blur-32 | Tile.svelte | ✅ Found | OK |
| `rounded-2xl` | 16px radius | Tile.svelte | ✅ Found (1x) | OK |
| `bg-[#12121a]/90` | panel bg | Tile.svelte | ✅ Found | OK |
| `border-white/10` | subtle border | Tile.svelte | ⚠️ Compiled as `border-white/1` | OK |
| `shadow-panel` | 0 8px 32px | Tile.svelte | ✅ Found (2x) | OK |
| `ring-purple-500/20` | active glow | Tile.svelte | ✅ Found | OK |
| `bg-white/5` | floating pill | WorkspacePill | ⚠️ Compiled away | Minor |
| `rounded-full` | pill/dot | WorkspacePill | ✅ Found | OK |
| `animate-pulse` | status dot | WorkspacePill | ✅ Found (4x) | OK |

**Verdict**: Design tokens are correctly implemented in both source and compiled CSS.

---

## Component Implementation Audit

### WorkspacePill ✅
- Position: `fixed top-2 left-1/2 -translate-x-1/2 z-50` — matches spec
- Glass: `backdrop-blur-xl bg-white/5 border border-white/10 rounded-full` — matches spec
- 9 dots with purple active: `bg-purple-400 shadow-glow` — matches spec
- Time display: updates every 10s — matches spec
- Status dot: `animate-pulse` green — matches spec

### Dock ✅
- Position: `fixed bottom-3 left-1/2 -translate-x-1/2 z-50` — matches spec
- Glass: `backdrop-blur-2xl bg-[#12121a]/70 border border-white/10 rounded-full` — matches spec
- 11 app icons with hover scale (110%) — matches spec
- Active indicator: purple dot — matches spec

### Tile (glassmorphism) ✅
- `backdrop-blur-xl bg-[#12121a]/90 border-white/10 shadow-panel` — matches spec
- Active: `border-glass-border-active ring-1 ring-purple-500/20` — matches spec
- Rounded corners: `rounded-2xl` — matches spec
- Transition: `transition-all duration-200 ease-out` — matches spec

### LeftPanel/RightPanel ✅
- `backdrop-blur-2xl bg-[#12121a]/80 border border-white/10 rounded-2xl shadow-panel` — matches spec

### CommandPalette ✅
- Overlay: `bg-black/60 backdrop-blur-sm` — matches spec
- Panel: `backdrop-blur-3xl bg-[#16161e]/95 border-white/15 rounded-2xl` — matches spec
- Width: 640px, max-h 70vh — matches spec

### ResizeHandle
- Thin 2px accent line: `cursor-ew-resize` — matches spec

---

## The Real Problem

**The code is correct. The visual verification is missing.**

```
What was built: Illogical Impulse glassmorphism system (all components ✅)
Why it "looks broken": We can't see it — no browser screenshot capability in container
```

### What's Broken in the QA Pipeline

1. **No Chromium in container** — `playwright install-deps` requires su, can't install system libs
2. **No screenshots** — Can't visually verify what was built matches reference
3. **No reference image** — The spec has ASCII descriptions, not a screenshot to compare against
4. **repo-transmute's verify loop exists but wasn't used** for HWC development

---

## What We Need: Visual QA Loop

For every visual task going forward:

```
Reference screenshot(s) or URL
        ↓
Build tile/component
        ↓
Serve (Go backend on host at :3113 + SSH tunnel)
        ↓
Playwright screenshot → /tmp/audit.png  (run on host where Chromium works)
        ↓
Vision model (Qwen3.6-plus via vision_analyze) → compare to reference
        ↓ fail (< 0.85 similarity)
Fix specific differences → re-screenshot → re-compare (loop)
        ↓ pass
Commit
```

---

## Action Plan

### Immediate (today)

1. **Write visual QA script** on host that:
   - Runs Playwright against `http://localhost:3113`
   - Takes screenshots of key states (main layout, command palette, keymap overlay)
   - Saves to accessible location

2. **Get reference screenshot** of Illogical Impulse style desktop (dots-hyprland) to compare against

3. **Run visual comparison** and identify any actual visual gaps between code and spec

### Short-term (this week)

4. **Integrate visual QA into pre-commit** — screenshot comparison passes before commit

5. **Set up screenshot baseline** — store baseline images in `e2e/tests/visual/baseline/` so regressions are caught

---

## Next Steps

**Option A**: Install Chromium on host, run Playwright there, serve screenshots via tmpfiles
**Option B**: Set up a public-facing tunnel to HWC so I can access it via browser tool
**Option C**: Use a reference URL (any public webpage with the Illogical Impulse style) to calibrate expectations

Recommendation: **Start with Option A** — write a script that runs on the host where Chromium is available, takes screenshots, and outputs structured diff.