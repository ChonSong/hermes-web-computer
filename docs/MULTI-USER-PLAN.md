# MULTI-USER-PLAN.md — Team Support for hermes-web-computer

> Architecture for adding multi-user/team support to HWC.

## Context

HWC currently runs as a **single-user** system. The WebSocket multiplexer handles one browser connection at a time, with a single `m.layout` shared across connections. Session and workspace state are per-user.

The goal is to support **multiple concurrent users**, each with their own workspace, session, and layout tree — while sharing access to shared tiles, files, and the Hermes Agent.

---

## Design Principles

1. **HWC shell per user** — Each user gets their own HWC shell (Waybar, dock, panels, tiles)
2. **Coder for workspace lifecycle** — Coder provisions and manages per-user Linux VMs (workspaces)
3. **Shared tiles** — Certain tiles (e.g., shared code editor, group chat) can be collaborative
4. **OIDC auth** — Keycloak for identity + access management
5. **Backend-owned truth** — Same model as single-user; each user's backend owns their state

---

## Architecture

### User Model

```
User → OIDC (Keycloak) → HWC Shell → Backend session → Coder workspace
```

| Entity | Description |
|--------|-------------|
| **User** | Authenticated via OIDC (Keycloak). Has a user ID, display name, avatar. |
| **HWC Shell** | The browser-based UI. Each user has their own shell instance. |
| **Backend Session** | Per-connection WebSocket session. Tracks workspace, layout, focused tile. |
| **Coder Workspace** | Per-user Linux VM. Provisioned on first login, suspended on logout. |

### Workspace Mapping

| HWC Concept | Coder Concept | Notes |
|-------------|---------------|-------|
| HWC Workspace 1-9 | Coder workspace +9 offset | HWC workspace 1 = Coder workspace `hwc-{user}-1` |
| HWC Layout Tree | Coder workspace state | Persisted to Coder workspace filesystem |
| HWC Session | Coder template | Pre-provisioned with user's tools and config |
| HWC Dock apps | Coder templates | Per-user app templates |

### Coder Integration Points

```go
// backend/coder/manager.go
type WorkspaceManager struct {
    endpoint string  // Coder API endpoint (e.g., http://localhost:7080/api/v2)
    apiKey   string  // Coder API key (from config)
    template string  // HWC workspace template ID
}

func (m *WorkspaceManager) Provision(ctx context.Context, userID string) (*Workspace, error)
func (m *WorkspaceManager) Suspend(ctx context.Context, workspaceID string) error
func (m *WorkspaceManager) Delete(ctx context.Context, workspaceID string) error
func (m *WorkspaceManager) GetState(ctx context.Context, workspaceID string) (*WorkspaceState, error)
```

### OIDC Flow

```
Browser → HWC → Keycloak (OIDC provider)
         ←─── token ───

HWC validates token on every WebSocket connection
Token contains: user_id, email, display_name, roles
Roles: admin (full access), user (own workspace), viewer (read-only)
```

### Multi-User WebSocket Architecture

```go
// backend/ws/multi.go
type MultiMultiplexer struct {
    sessions map[string]*UserSession  // userID → session
    mux      *Multiplexer              // delegate to for single-user packages
    coder    *WorkspaceManager         // Coder integration
    auth     *OIDCValidator            // Keycloak token validation
}

type UserSession struct {
    userID     string
    ws         *websocket.Conn
    workspace  string  // Coder workspace ID
    layout     *layout.Tree
    focusCtx   string  // focused tile ID
}
```

**Key changes:**
- `Multiplexer.HandleWebSocket` now takes `userID` from OIDC token
- `layout.Tree` is per-user-session, not global singleton
- `session.Store` is per-user-directory: `~/.hermes/hermes-web-computer/users/{userID}/sessions/`
- `workspaceStore` persisted per-user: `~/.hermes/hermes-web-computer/users/{userID}/workspace.json`

### Shared Tiles (Future)

For collaborative tiles (shared code editor, group chat):

```
┌─────────────────────────────────────────────────────────┐
│               Shared Tile Protocol                       │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  User A ──┐                                             │
│  User B ──┼─── WebSocket ── HWC Backend ── Coder CRDT  │
│  User N ──┘         │                    │               │
│                     ▼                    ▼               │
│              Shared Tile State      Sync Engine          │
│              (per-tile)             (per-workspace)     │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

Shared tile state uses operational transformation — similar to how Google Docs handles collaborative editing. Coder provides the infrastructure for CRDT-based state sync.

---

## Implementation Phases

### Phase M0: Multi-User Infrastructure (Foundation)

**Goal:** Make backend support multiple concurrent sessions

Tasks:
- [ ] Add `userID` field to WebSocket connection context
- [ ] Move `layout.Tree` from global singleton to per-session
- [ ] Move `session.Store` from global path to per-user path
- [ ] Add OIDC token validation middleware
- [ ] Add `WorkspaceManager` interface for Coder
- [ ] Add config: `multi_user.enabled`, `coder.endpoint`, `coder.api_key`, `oidc.*`
- [ ] Add user registry: `~/.hermes/hermes-web-computer/users/`
- [ ] Add per-user workspace persistence: `users/{userID}/workspace.json`

**Backend changes:**
```go
// New config fields (config.yaml)
multi_user:
  enabled: true
  oidc:
    issuer: "https://keycloak.example.com"
    client_id: "hwc"
    client_secret: "..."
  coder:
    endpoint: "http://localhost:7080/api/v2"
    api_key: "..."
    template: "hwc-default"

# Per-user directories
users/{userID}/
├── sessions/        # session store
├── workspace.json  # workspace state
└── config.yaml     # user-specific overrides
```

### Phase M1: Keycloak OIDC Integration

**Goal:** Authenticate users via Keycloak OIDC

Tasks:
- [ ] Add OIDC discovery (fetch `.well-known/openid-configuration`)
- [ ] Add JWT validation (RS256, audience, issuer, expiry)
- [ ] Add Keycloak client for user info + token introspection
- [ ] Add login page with Keycloak redirect
- [ ] Add token refresh flow
- [ ] Add logout + token revocation
- [ ] Add user info to WebSocket connection (display name, avatar)

**Keycloak setup:**
```bash
# Create realm: hwc
# Create client: hwc-client (confidential, standard flow)
# Create mapper: user_id → sub claim, email → email claim, name → name claim
# Create users: testuser1, testuser2
```

### Phase M2: Coder Workspace Lifecycle

**Goal:** Provision per-user Linux VMs via Coder

Tasks:
- [ ] Add Coder API client (Go SDK or direct REST)
- [ ] Add workspace provision on first login
- [ ] Add workspace suspend on logout
- [ ] Add workspace delete (admin only)
- [ ] Add workspace state sync (layout tree ↔ Coder volume)
- [ ] Add workspace templates (default, dev, minimal)
- [ ] Add workspace sharing (admin can view all, user sees own)

**Coder workspace templates:**
```yaml
# template.yaml for HWC workspace
name: hwc-default
instances: ["create", "stop"]
image: ubuntu:latest
params:
  - name: user_id
    description: HWC user ID
  - name: disk_size
    default: "20GB"
resources:
  cpu: 2
  memory: 4G
  disk: 20GB
```

### Phase M3: Shared Tiles (Collaborative)

**Goal:** Enable real-time collaboration on shared tiles

Tasks:
- [ ] Add shared tile registry (`shared_tiles` table)
- [ ] Add CRDT-based state sync for shared tiles
- [ ] Add presence indicators (who's viewing which tile)
- [ ] Add conflict resolution (last-write-wins for simple tiles, OT for text)
- [ ] Add shared tile access control (owner, editor, viewer)
- [ ] Add audit log (who changed what)

**Shared tile types:**
| Tile | Sync Strategy | Conflict Resolution |
|------|--------------|---------------------|
| Group Chat | Append-only log | None (append only) |
| Shared Code Editor | CRDT text | Operational transformation |
| Shared Terminal | Leader-follower | Leader wins (one editor at a time) |
| Shared File Browser | Git-like | Last-write-wins with conflict notification |

### Phase M4: Access Control + Audit

**Goal:** Role-based access + audit trail

Tasks:
- [ ] Add RBAC: admin, user, viewer roles
- [ ] Add tile-level permissions (who can open which tiles)
- [ ] Add workspace-level permissions (who can access which workspaces)
- [ ] Add audit log: all actions with user, timestamp, resource
- [ ] Add audit log viewer in UI (admin only)
- [ ] Add session recording (optional, for compliance)

---

## Open Questions

1. **Workspace isolation vs sharing:** Should users be able to share their workspace with another user (read-only or collaborative)? This adds significant complexity — start with fully isolated.

2. **Coder vs plain VMs:** Coder provides workspace lifecycle + OIDC + audit. Could instead use plain LXD containers + SSH + manual config. Coder wins on UX and lifecycle management, but adds dependency.

3. **Shared tile CRDT vs leader-follower:** CRDT is theoretically correct but complex to implement. Leader-follower is simpler but requires coordination. Start with leader-follower for most tiles, CRDT only for text editing.

4. **Session persistence across logout:** If a user logs out and back in, should their workspace state (open tiles, layout) be restored? Yes — this is the main UX win. Store layout in Coder workspace, restore on login.

5. **Offline support:** Should users be able to work offline (no connection to HWC backend)? This violates "backend-owned truth" and adds CRDT complexity. Not in scope for v1.

---

## Success Criteria

| Criterion | Target | How Measured |
|-----------|--------|---------------|
| Concurrent users | 10+ | Load test with 10 simultaneous WebSocket connections |
| Login latency | <2s | OIDC redirect → HWC shell rendered |
| Workspace provision | <30s | Time from first login to Coder workspace ready |
| Workspace suspend | <5s | Time from logout to Coder workspace suspended |
| Auth correctness | 100% | All WebSocket connections must have valid OIDC token |
| Isolation | 100% | User A cannot read/write User B's sessions or workspace |

---

## Related Docs

- `docs/ARCHITECTURE.md` — Backend architecture (multiplexer, session, layout)
- `docs/WAYBAR-SPEC.md` — Waybar + workspace UI spec
- `~/.hermes/skills/hermes-computer/SKILL.md` — Operations guide
- pad.ws (github.com/coderamp-labs/pad.ws) — Reference for whiteboard+Coder integration