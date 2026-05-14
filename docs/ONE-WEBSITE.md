# One Website to Rule Them All

> Complete architecture analysis: how agent-os, hermes-agent, and hermes-computer fit together

**Generated:** 2026-05-11
**Status:** 10% complete — core architecture proven, integrations pending

---

## 1. The Three Pillars

### 🧠 hermes-agent (The Brain)
- **What:** Python AI agent with skills, tools, memory, multi-platform gateway
- **Status:** ✅ 100% working, running on host port 8642
- **Role:** Provides AI intelligence to ALL other systems
- **Don't touch:** It's working perfectly. Just wire it in.

### 📊 agent-os (The Legacy Dashboard)
- **What:** React + Express dashboard with 22 pages, 11 themes, PostgreSQL
- **Status:** ✅ 100% working, running locally on port 3001
- **Role:** CURRENT product — will become TILES inside hermes-web-computer
- **Key pages to migrate:** Agent Status, Session History, System Metrics

### 🖥️ hermes-web-computer (The Future Shell)
- **What:** Go backend + Svelte 5 SPA with tiling interface
- **Status:** 🔄 70% complete, core architecture proven (E2E test passes)
- **Role:** NEW frontend shell — ALL apps become tiles inside it
- **Missing:** Hermes integration, LiteLLM, voice, browser, sandbox tiles

---

## 2. The Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     hermes-web-computer (THE SHELL)                 │
│                                                                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │
│  │ Terminal │ │ Browser  │ │ Dashboard│ │ Voice    │ │ Code     │ │
│  │   Tile   │ │   Tile   │ │   Tile   │ │   Tile   │ │  Edit    │ │
│  │  (✅)    │ │ (bytebot)│ │(agent-os)│ │(Fun-Audio│ │ (Monaco) │ │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ │
│       │             │            │             │             │      │
│       └─────────────┴────────────┴─────────────┴─────────────┘      │
│                            │                                        │
│                  WebSocket (JSON-RPC)                                │
│                            │                                        │
│              ┌─────────────▼─────────────┐                          │
│              │   Go Backend Multiplexer  │                          │
│              │  - Layout engine           │                          │
│              │  - PTY supervisor          │                          │
│              │  - Security enforcer       │                          │
│              │  - Telemetry               │                          │
│              └─────────────┬─────────────┘                          │
└────────────────────────────┼────────────────────────────────────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
  ┌─────────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
  │  hermes-agent  │ │  Docker     │ │  External   │
  │  (THE BRAIN)   │ │  Sandbox    │ │  APIs       │
  │                │ │  (cua/      │ │  (LLMs,     │
  │ - Skills       │ │   bytebot)  │ │   Web, etc) │
  │ - Tools        │ │             │ │             │
  │ - Memory       │ │             │ │             │
  │ - LLM Routing  │ │             │ │             │
  │ - Platforms    │ │             │ │             │
  └────────────────┘ └─────────────┘ └─────────────┘
```

**The flow:**
1. User opens hermes-web-computer → sees tiling interface
2. User splits tiles → Terminal, Browser, Dashboard, Voice appear
3. Each tile talks to Go backend via WebSocket (JSON-RPC)
4. Go backend routes to Hermes Agent for AI, Docker for sandbox, APIs for data
5. Hermes uses skills, tools, memory to respond
6. Responses flow back to tiles via WebSocket

---

## 3. Current Progress

| Component | Status | Progress | Effort Remaining |
|-----------|--------|----------|------------------|
| Tiling Interface | ✅ Working | 100% | 0 |
| Terminal Tile | ✅ Working | 100% | 0 |
| PTY Supervisor | ✅ Working | 100% | 0 |
| Layout Engine | ✅ Working | 100% | 0 |
| Security Enforcer | ✅ Working | 100% | 0 |
| Telemetry | ✅ Working | 100% | 0 |
| Docker Compose | ✅ Working | 100% | 0 |
| E2E Test | ✅ Passing | 100% | 0 |
| CI/CD | ✅ Working | 100% | 0 |
| **Hermes Integration** | ❌ TODO | 0% | 2 days |
| **LiteLLM Adapter** | ❌ TODO | 0% | 3 days |
| **Fun-Audio-Chat Bridge** | ❌ Stub | 10% | 2 days |
| **Monaco Editor Tile** | ❌ Stub | 10% | 1 day |
| **Browser Tile** | ❌ Not started | 0% | 5 days |
| **Dashboard Tile** | ❌ Not started | 0% | 3 days |
| **Sandbox Tile** | ❌ Not started | 0% | 5 days |
| **Multi-User** | ❌ Not started | 0% | 3 days |
| **Vision Testing** | ❌ Not started | 0% | 2 days |

**Overall: ~50% of core done, ~10% of vision done**

---

## 4. The Migration Path

### Phase 1: Complete Core (2 weeks)
1. **Hermes Agent Integration** — Wire tool.execute to Hermes tools (2 days)
2. **LiteLLM Adapter** — Multi-provider model switching (3 days)
3. **Fun-Audio-Chat Bridge** — Full voice integration (2 days)
4. **Monaco Editor Tile** — Code editing in tiling interface (1 day)

### Phase 2: Migrate agent-os Pages (1 week)
5. **Dashboard Tile** — Top 3 agent-os pages as Svelte 5 tiles (3 days)
   - Agent Status Tile (from ModelsPage)
   - Session History Tile (from SessionsPage)
   - System Metrics Tile (from AnalyticsPage)

### Phase 3: Computer-Use Tiles (2 weeks)
6. **Browser Tile** — Migrate bytebot screenshot capture + DOM analysis (5 days)
7. **Sandbox Tile** — Migrate cua screen capture + input routing (5 days)
8. **File Manager Tile** — Build from scratch (2 days)

### Phase 4: Polish & Production (1 week)
9. **Multi-User Sessions** — Concurrent isolated workspaces (3 days)
10. **Vision Testing** — Automated visual verification (2 days)
11. **Performance Optimization** — <100ms interrupt, <50ms layout (2 days)

**Total: 6 weeks to v1.0 (4 core tiles working)**

---

## 5. What Makes This Work

### ✅ Strengths
- **hermes-web-computer core is proven** — E2E test passes, architecture is solid
- **hermes-agent works perfectly** — No need to rebuild, just wire it in
- **agent-os has 22 working pages** — Migration source is real and tested
- **repo-transmute v2 exists** — Migration engine is built and tested
- **seans-reporepo catalog** — 34 owned + 133 starred repos indexed for migration

### ❌ Risks
- **Scope creep** — Trying to build everything at once will fail
- **Svelte 5 immaturity** — Ecosystem is young, may need fallbacks
- **agent-os migration complexity** — 22 React pages → Svelte 5 is non-trivial
- **Resource limits** — Multiple tiles + sandbox could exhaust memory/CPU

### 🎯 Strategy
1. **Finish the core first** — Don't add tiles until shell works
2. **Migrate incrementally** — Top 3 pages first, prove the pattern
3. **Use repo-transmute** — Don't migrate by hand
4. **Stay lean** — 4 core tiles for v1.0, rest later
5. **Vision test everything** — Automated verification prevents regression

---

## 6. Repo Map

| Repo | Purpose | Stack | Status | Lines |
|------|---------|-------|--------|-------|
| **hermes-web-computer** | Tiling AI desktop shell | Go + Svelte 5 | 🔄 70% | ~1,700 |
| **agent-os** | Legacy dashboard (migration source) | React + Express | ✅ 100% | ~17,000 |
| **hermes-agent** | AI brain (stays separate) | Python + FastAPI | ✅ 100% | ~10,000 |
| **repo-transmute** | Migration engine | Python + Click | ✅ v2 complete | ~5,300 |
| **features-list** | Feature catalog | Markdown + HTML | ✅ Complete | ~9,900 |
| **hermes-computer-planning** | Planning docs | Markdown | ✅ Complete | ~26,000 |
| **seans-reporepo** | Repo catalog | YAML + Python | ✅ Complete | ~5,000 |

---

## 7. Next Steps

1. **Complete P0 tasks** in hermes-web-computer (Hermes + LiteLLM)
2. **Migrate top 3 agent-os pages** to Svelte 5 tiles
3. **Build Browser Tile** from bytebot extraction
4. **Wire Fun-Audio-Chat** for voice tile
5. **Add Monaco editor** tile
6. **Test end-to-end** with all 4 core tiles working
7. **Deploy to production**

**Estimated time to v1.0:** 6 weeks
**Estimated time to "one website to rule them all":** 8-10 weeks

---

*This is the definitive architecture document. All other planning docs feed into this vision.*
