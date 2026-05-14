# Hermes-Web-Computer: Application & Migration Plan

> Beyond completion-plan.md — what tiles to build, what starred repos to mine, and how repo-transmute makes it happen.

**Generated:** 2026-05-11  
**Stack Target:** Go backend + Svelte 5 SPA (hermes-web-computer)  
**Source Repos:** 34 owned + 133 starred, filtered to 12 migration candidates  
**Migration Engine:** repo-transmute v2 (vision-driven AST extraction → LLM migration → verification)

---

## 1. Tile Architecture

hermes-web-computer is a **tiling AI desktop**. Each tile is a Svelte 5 component backed by a Go handler. The layout engine manages split/resize/focus. Tiles communicate through the JSON-RPC multiplexer over WebSocket.

### Core Tiles (v1.0 — must ship)

| Tile | Source | Complexity | Priority |
|------|--------|------------|----------|
| **Terminal** | hermes-web-computer (built-in) | ✅ Done | v1.0 |
| **Browser** | bytebot (migrate) | High | v1.0 |
| **Voice Chat** | Fun-Audio-Chat (owned) | Medium | v1.0 |
| **Dashboard** | agent-os (migrate from React→Svelte) | Medium | v1.0 |

### Enhancement Tiles (v1.1)

| Tile | Source | Complexity | Priority |
|------|--------|------------|----------|
| **Code Editor** | Monaco.svelte (stub) | Low | v1.1 |
| **Research** | context7 + karpathy-skills | Low | v1.1 |
| **Sandbox** | cua (computer-use infra) | High | v1.1 |
| **Media** | yt-dlp (owned) | Low | v1.1 |

### Future Tiles (v1.2+)

| Tile | Source | Complexity | Priority |
|------|--------|------------|----------|
| **File Manager** | Build from scratch | Medium | v1.2 |
| **Kasm Workspace** | kasm-mcp-server-v2 | Medium | v1.2 |
| **Dotfiles Config** | sean-dotfiles (owned) | Low | v1.2 |

---

## 2. Migration Candidates (Ranked)

Each candidate is a starred repo that should be extracted, migrated, and integrated as a tile or component in hermes-web-computer.

### Tier 1: Direct Tile Material

#### 2.1 bytebot-ai/bytebot → Browser Tile
- **Stars:** 11,003 | **License:** Apache-2.0 | **Language:** TypeScript
- **What it is:** Full desktop AI agent with browser automation, screenshot capture, DOM analysis
- **What to extract:**
  - Browser screenshot capture → Svelte image component with live refresh
  - DOM analysis pipeline → Go handler that exposes DOM structure as JSON-RPC tool
  - Click/typing automation → Go sandbox input routing
- **Transpile needs:** TS→Go (backend handlers), TS→Svelte 5 (browser viewer)
- **repo-transmute plan:** `v2 migrate bytebot hermes-web-computer --extract browser-capture,dom-analysis --target svelte5+go`
- **Risk:** bytebot is a full desktop agent, not just browser. Need surgical extraction, not wholesale migration.
- **Effort:** 3-5 days

#### 2.2 trycua/cua → Sandbox Tile
- **Stars:** 15,833 | **License:** MIT | **Language:** Python/Swift/TS
- **What it is:** Cross-OS computer-use infrastructure (macOS mouse/keyboard + screen capture)
- **What to extract:**
  - Screen capture pipeline → Go screenshot service
  - Mouse/keyboard event routing → Go input dispatcher
  - Coordinate mapping → Layout engine integration
- **Transpile needs:** Python→Go (system calls), Swift/TS→ignore (macOS-only parts irrelevant)
- **repo-transmute plan:** `v2 migrate cua hermes-web-computer --extract screen-capture,input-routing --target go`
- **Risk:** cua is heavily macOS-focused. Linux adaptation needed for Docker sandbox.
- **Effort:** 3-4 days

#### 2.3 agent-os Dashboard → Dashboard Tile (React→Svelte 5)
- **Stars:** 0 (owned) | **License:** — | **Language:** React + Express
- **What it is:** Your live dashboard (22 pages, 11 themes, PostgreSQL backend)
- **What to extract:**
  - Agent status page → Svelte tile
  - Session history → Svelte tile with Go backend query
  - System metrics → Svelte tile pulling from telemetry
- **Transpile needs:** React→Svelte 5 (components), Express→Go (API routes already exist in hermes-web-computer)
- **repo-transmute plan:** `v2 migrate agent-os hermes-web-computer --extract dashboard-pages --target svelte5`
- **Risk:** 22 pages is a lot. Prioritize: agent status, session list, system metrics.
- **Effort:** 2-3 days (for top 3 pages)

### Tier 2: Component Libraries

#### 2.4 sveltejs_ai-tools → AI Tool Components
- **Stars:** new | **License:** — | **Language:** TypeScript
- **What it is:** AI tooling primitives for Svelte apps
- **What to extract:** Any reusable Svelte components (chat UI, streaming response display, tool call cards)
- **Transpile needs:** Minimal — already Svelte, may need Svelte 4→5 rune updates
- **Effort:** 1-2 days

#### 2.5 upstash/context7 → Research Tile Data Source
- **Stars:** new | **License:** — | **Language:** TypeScript
- **What it is:** Code snippet retrieval / developer knowledge base
- **What to extract:** API integration → Go service that queries Context7 and returns results as tile content
- **Transpile needs:** TS API client → Go HTTP client
- **Effort:** 1 day

#### 2.6 sindresorhus/awesome → Curated Resource Index
- **Stars:** 336K | **License:** MIT | **Language:** Markdown
- **What it is:** The canonical awesome list
- **What to extract:** Not code — use as a resource for building the Research tile's knowledge base
- **Transpile needs:** None — data only
- **Effort:** 0.5 days (import and index)

### Tier 3: Reference / Inspiration

#### 2.7 Front-End-Checklist / design-resources → Design Reference
- **Not migration candidates** — use as design QA during tile development
- Tag these in catalog as `reference` not `candidate`
- Includes: `Front-End-Checklist`, `design-resources-for-developers`, `frontend-dev-bookmarks`, `awesome-frontend-resources`, `omatsuri`

#### 2.8 coder-desktop-linux → Architecture Inspiration Only
- Already analyzed in existing hermes-computer-planning README
- The VPN/workspace connectivity *concept* is worth adopting; the C# code is not
- Build from scratch in Go with WireGuard/nhooyr.io/websocket

---

## 3. repo-transmute Integration Strategy

### 3.1 Ingestion Pipeline

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐     ┌──────────────┐
│ Starred     │───▶ │ repo-        │───▶ │ AST Blueprint │───▶ │ Migration    │
│ Repo        │     │ transmute    │     │ + Screenshots │     │ Plan         │
│ (bytebot)   │     │ v2 ingest    │     │               │     │ (candidates/)│
└─────────────┘     └──────────────┘     └───────────────┘     └──────────────┘
```

1. **Ingest:** `repo-transmute v2 ingest <repo>` — clone, detect framework, extract AST
2. **Blueprint:** Get component tree, API patterns, style tokens
3. **Plan:** Generate migration plan (what to extract, what to skip)
4. **Migrate:** `repo-transmute v2 migrate` — LLM-driven transpilation
5. **Verify:** Vision scoring — screenshot source vs migrated output
6. **Heal:** Auto-fix if vision score < threshold

### 3.2 Catalog → Candidates Pipeline

The seans-reporepo catalog feeds repo-transmute:

```
seans-reporepo COMBINATORIAL.md
    │
    ├── Tags overlap: "browser-automation" + "svelte"
    │   └── → bytebot → Browser Tile candidate
    │
    ├── Tags overlap: "computer-use" + "go"  
    │   └── → cua → Sandbox Tile candidate
    │
    └── Tags overlap: "dashboard" + "svelte"
        └── → agent-os pages → Dashboard Tile candidate
```

### 3.3 What repo-transmute Needs to Know

For each migration, the engine needs:
1. **Source repo** (GitHub URL + branch)
2. **Extraction scope** (which files/components)
3. **Target framework** (Svelte 5 + Go)
4. **Style system** (Tailwind? CSS modules? — hermes-web-computer uses Tailwind)
5. **Verification URL** (how to screenshot the source for comparison)

---

## 4. Tile Component Specs

### 4.1 Browser Tile

```
┌─────────────────────────────────────────┐
│  Browser Tile (Svelte 5)                │
├─────────────────────────────────────────┤
│  ┌───────────────────────────────────┐  │
│  │ Screenshot Stream (500ms refresh) │  │
│  │ ┌───────────────────────────────┐ │  │
│  │ │  [web page rendering]         │ │  │
│  │ └───────────────────────────────┘ │  │
│  └───────────────────────────────────┘  │
│  URL: [https://example.com      ] [Go]  │
│  [⬅] [➡] [↻] [🔍] [🤖 AI Act]           │
└─────────────────────────────────────────┘

Props: { url: string, sandboxId: string, aiMode: boolean }
Events: on:navigate, on:ai-action, on:screenshot
Backend: Go sandbox manager + bytebot browser engine
```

### 4.2 Voice Chat Tile

```
┌─────────────────────────────────────────┐
│  Voice Chat Tile (Svelte 5)             │
├─────────────────────────────────────────┤
│                                         │
│        ╭─────────────╮                  │
│        │  [waveform]  │                  │
│        │   ● ● ● ● ●  │                  │
│        ╰─────────────╯                  │
│                                         │
│  Status: Listening / Speaking / Idle    │
│  [🎤 Mic] [🔊 Speaker] [⏹ Stop]          │
│                                         │
│  Transcript:                            │
│  > "Show me the browser tile"           │
└─────────────────────────────────────────┘

Props: { audioBridgeUrl: string, sessionId: string }
Events: on:transcript, on:interrupt, on:status-change
Backend: audio/bridge.go + Fun-Audio-Chat WebSocket
```

### 4.3 Dashboard Tile

```
┌─────────────────────────────────────────┐
│  Dashboard Tile (Svelte 5)              │
├─────────────────────────────────────────┤
│  ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │ Agents  │ │ Sessions│ │ System  │   │
│  │  ● 2    │ │ 3 active│ │ 42% CPU │   │
│  └─────────┘ └─────────┘ └─────────┘   │
│                                         │
│  Agent Status:                          │
│  ┌───────────────────────────────────┐  │
│  │ hermes-agent  │ ● running │ 12s  │  │
│  │ audio-agent   │ ● running │ 3m   │  │
│  │ browser-agent │ ○ stopped │ -    │  │
│  └───────────────────────────────────┘  │
│                                         │
│  [Refresh] [Logs] [Restart]             │
└─────────────────────────────────────────┘

Props: { apiUrl: string, refreshInterval: number }
Backend: Go health endpoint + agent-os API routes (migrated)
```

---

## 5. Cron & Automation Plan

### 5.1 Tier 1: Catalog Refresh (Weekly)
- **Schedule:** Monday 09:00 (existing)
- **Action:** Pull latest stars, regenerate catalog, compute changelog
- **Output:** README.md, COMBINATORIAL.md, per-repo .md files
- **No action taken** — just updates the index

### 5.2 Tier 2: Candidate Alert (On New Star)
- **Trigger:** When a new star matches combinatorial criteria (bridges owned + starred with ≥3 tags)
- **Action:** Generate a `candidates/<repo>.md` profile, add to COMBINATORIAL.md
- **Output:** New candidate file in seans-reporepo
- **Delivery:** Hermes notifies Sean: "New migration candidate: <repo> — matches <tags>"

### 5.3 Tier 3: Auto-Ingest (Manual Gate)
- **Trigger:** Sean says "ingest <repo>" or confirms a candidate alert
- **Action:** Run `repo-transmute v2 ingest <repo>`, generate blueprint
- **Output:** Blueprint in repo-transmute/data/
- **Delivery:** Hermes reports component count, migration estimate

### 5.4 Tier 4: Auto-Migrate (Explicit Approval Only)
- **Trigger:** Sean says "migrate <repo> to <tile>"
- **Action:** Full repo-transmute pipeline with vision verification
- **Output:** Migrated code in hermes-web-computer/src/
- **Delivery:** PR with vision score, before/after screenshots

**Safety:** Tiers 3 and 4 never run autonomously. Sean must approve.

---

## 6. Risk Register

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| bytebot API changes during migration | Medium | High | Pin to specific commit, extract core only |
| repo-transmute vision scoring fails on complex pages | Medium | Medium | Lower threshold, manual verification fallback |
| Svelte 5 rune migration from Svelte 4/React breaks components | High | Medium | Incremental migration, test each component |
| Go sandbox + Docker resource limits too tight | Low | High | Start generous, tighten after profiling |
| Fun-Audio-Chat protocol undocumented changes | Low | Medium | Version pin the bridge, document protocol |
| Sean stars a repo that's actually dead/archived | Medium | Low | Catalog already filters archived repos |

---

## 7. Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-05-11 | Target stack: Go + Svelte 5 | Newer spec, leaner deploys, hermes-web-computer architecture |
| 2026-05-11 | agent-os is migration source | Legacy stack, 22 React pages worth migrating |
| 2026-05-11 | bytebot = Browser Tile | Best-in-class browser automation, Apache-2.0 license |
| 2026-05-11 | cua = Sandbox Tile | Cross-platform computer-use, MIT license, actively maintained |
| 2026-05-11 | 4 cron tiers, auto-ingest gated | Balance automation with safety — Sean approves migrations |
| 2026-05-11 | catalog feeds repo-transmute | Catalog is the input, repo-transmute is the engine |
| 2026-05-11 | 4 core tiles for v1.0 | Terminal + Browser + Voice + Dashboard = complete workflow |

---

## 8. Quick Start Commands

```bash
# Refresh catalog
cd /home/sean/.hermes/cache/seans-reporepo && bash scripts/refresh.sh

# Ingest a candidate repo
cd /opt/data/repo-transmute-v2
python3 -m src.cli v2 ingest https://github.com/bytebot-ai/bytebot --output data/bytebot-blueprint

# Migrate to Svelte 5
python3 -m src.cli v2 migrate data/bytebot-blueprint /opt/data/hermes-web-computer \
  --framework typescript --target-stack svelte5+go \
  --extract browser-capture,dom-analysis

# Run vision verification
python3 -m src.cli v2 verify data/bytebot-blueprint /opt/data/hermes-web-computer
```

---

*This document supersedes the ad-hoc planning. The completion-plan.md covers the technical buildout; this document covers the strategic migration from starred repos into real tile applications.*
