# Xpra Integration — Native GUI Escape Hatch

> **Purpose:** Enable running arbitrary native Linux GUI applications (Firefox, JetBrains IDEs, LibreOffice, etc.) inside hermes-web-computer tiles when web-native alternatives don't exist or aren't feasible.

---

## 1. Why Xpra

hermes-web-computer is designed around web-native tiles (Svelte + Go). However, many professional tools have no viable web replacement:

| Tool | Why No Web Tile |
|------|-----------------|
| Firefox / Chrome | DevTools, browser automation, PDF rendering |
| JetBrains IDEs (IntelliJ, PyCharm) | Complex editor UI, JVM dependency |
| LibreOffice | Document rendering, VBA interop |
| Electron apps (VS Code, Slack) | Native menus, system tray |
| Wine / Windows apps | Zero web equivalent |

**Alternatives considered:**

| Option | Problem |
|--------|---------|
| VNC | High latency, no clipboard sync, no WebSocket native |
| X11 over SSH (`ssh -X`) | Unidirectional, no reconnection resilience |
| WebRTC | Complex infrastructure, firewall issues |
| NoVNC | Works but lacks persistent sessions, multi-window |

**Xpra wins because:**
- Sits on top of X11, works with any X app
- Supports WebSocket/HTML5 client (`xpra html5`)
- Persistent sessions (reattach, don't restart)
- Clipboard, Bell, File transfer built-in
- Mature, actively maintained (`github.com/Xpra-org/xpra`)
- Single port (9443+) for all sessions

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  hermes-web-computer Backend (Go)                           │
│  ┌──────────────┐  ┌─────────────────┐  ┌───────────────┐ │
│  │ ws Package   │  │ apps.go         │  │ xpra Package  │ │
│  │ (JSON-RPC)   │  │ (App Launcher)  │  │ (Manager)     │ │
│  └──────────────┘  └─────────────────┘  └───────────────┘ │
│         │                   │                  │         │
│         └───────────────────┼──────────────────┘         │
│                             │                             │
│  ┌─────────────────────────▼────────────────────────────┐ │
│  │  Xpra Server (spawned per user session)              │ │
│  │  Unix socket: /run/user/uid/xpra/$DISPLAY            │ │
│  │  WebSocket : 0.0.0.0:9443 (html5 client proxy)      │ │
│  └──────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                    WebSocket / TCP
                              │
            ┌─────────────────┼─────────────────┐
            │                 │                 │
      ┌─────▼─────┐    ┌─────▼─────┐    ┌──────▼───────┐
      │ Tile      │    │ Tile      │    │ Tile         │
      │ (Firefox) │    │ (IntelliJ)│    │ (LibreOffice)│
      └───────────┘    └───────────┘    └──────────────┘
```

### Session Model

Each user gets **one persistent Xpra server** per session. Applications are attached as windows to that server. The backend manages:

1. **Xpra server lifecycle** (start/stop/env vars)
2. **Window→tile mapping** (which Xpra window renders in which tile)
3. **WebSocket proxy** (frontend connects to Xpra's HTML5 client)

---

## 3. Backend Integration

### 3.1 `xpra/manager.go`

```go
package xpra

import (
    "context"
    "encoding/json"
    "fmt"
    "net"
    "os"
    "os/exec"
    "path/filepath"
    "sync"
    "time"

    "github.com/nhooyr/websocket"
)

// Manager handles the per-session Xpra server lifecycle.
type Manager struct {
    sessionID   string
    displayNum  int
    sockPath    string
    httpPort    int
    cmd         *exec.Cmd
    mu          sync.Mutex
    started     bool
}

// New creates a Manager for the given session.
func New(sessionID string, displayNum int) *Manager {
    uid := os.Geteuid()
    sockDir := fmt.Sprintf("/run/user/%d/xpra", uid)
    sockPath := filepath.Join(sockDir, fmt.Sprintf("%d", displayNum))
    httpPort := 9443 + displayNum
    return &Manager{
        sessionID:  sessionID,
        displayNum: displayNum,
        sockPath:   sockPath,
        httpPort:   httpPort,
    }
}

// Display returns the X display string (e.g., ":10").
func (m *Manager) Display() string {
    return fmt.Sprintf(":%d", m.displayNum)
}

// Start launches the Xpra server with session shadowing disabled.
// Uses mode "server" (no root window), starts html5 web bridge.
func (m *Manager) Start(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.started {
        return nil
    }

    sockDir := filepath.Dir(m.sockPath)
    if err := os.MkdirAll(sockDir, 0755); err != nil {
        return fmt.Errorf("create xpra socket dir: %w", err)
    }

    args := []string{
        "start",
        "--daemon",
        "--no-daemonize",
        fmt.Sprintf("--socket-dir=%s", sockDir),
        m.Display(),
        "--start=xterm",
        "--bind=TCP",
        fmt.Sprintf("--bind-web=%d", m.httpPort),
        "--html=on",
        "--idle-timeout=0",
        "--server-Idle-Timeout=0",
        // No session shadowing — we run a standalone server
    }

    cmd := exec.CommandContext(ctx, "xpra", args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("xpra start: %w", err)
    }

    m.cmd = cmd
    m.started = true

    // Wait for socket to appear
    return m.waitForSocket(5 * time.Second)
}

func (m *Manager) waitForSocket(timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if _, err := os.Stat(m.sockPath); err == nil {
            return nil
        }
        time.Sleep(100 * time.Millisecond)
    }
    return fmt.Errorf("xpra socket not created within %v", timeout)
}

// Stop terminates the Xpra server.
func (m *Manager) Stop() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.cmd != nil && m.cmd.Process != nil {
        m.cmd.Process.Kill()
    }
    m.started = false
    return exec.Command("xpra", "stop", m.Display()).Run()
}

// AttachWindow runs an application and attaches its windows to the Xpra server.
func (m *Manager) AttachWindow(ctx context.Context, app string, args []string) (*exec.Cmd, error) {
    env := os.Environ()
    env = append(env, fmt.Sprintf("DISPLAY=%s", m.Display()))

    cmd := exec.CommandContext(ctx, app, args...)
    cmd.Env = env
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("attach window %s: %w", app, err)
    }
    return cmd, nil
}

// WSEndpoint returns the WebSocket URL for the Xpra HTML5 client.
func (m *Manager) WSEndpoint() string {
    return fmt.Sprintf("ws://localhost:%d", m.httpPort)
}
```

### 3.2 `xpra/client.go` — WebSocket Proxy

The frontend cannot connect directly to Xpra's WebSocket (different protocol). A proxy in the Go backend relays:

```go
package xpra

import (
    "bufio"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"

    "github.com/nhooyr/websocket"
)

// ProxyHandler relays between the frontend WS (hermes protocol) and Xpra WS.
type ProxyHandler struct {
    mgr *Manager
}

// ServeWs upgrades a ui-protocol WS connection and proxies to Xpra.
func (p *ProxyHandler) ServeWs(w http.ResponseWriter, r *http.Request) {
    // Frontend sends: {"protocol":"xpra","method":"attach","params":{"window_id":"..."}}
    // Upgrade to raw WS, then pipe to Xpra's WS endpoint
    u := url.URL{Scheme: "ws", Path: "/"}
    xpraWS, _, err := websocket.Dial(r.Context(), u.String(), nil)
    if err != nil {
        http.Error(w, fmt.Sprintf("xpra dial: %v", err), 503)
        return
    }
    defer xpraWS.Close(websocket.StatusGoingAway, "done")

    clientWS, _, err := websocket.Accept(w, r)
    if err != nil {
        return
    }
    defer clientWS.Close(websocket.StatusGoingAway, "done")

    // Bidirectional relay
    go io.Copy(xpraWS, clientWS.Reader)
    io.Copy(clientWS, xpraWS.Reader)
}
```

> **Note:** A full Xpra protocol proxy is non-trivial. See Section 5 for simplification approach using Xpra's built-in HTML5 server.

---

## 4. Frontend Integration

### 4.1 `XpraTile.svelte`

The Xpra tile loads Xpra's HTML5 client in an `<iframe>` and handles window lifecycle:

```svelte
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  interface Props {
    windowId?: string;
    title?: string;
    app?: string;
  }

  let { windowId = '', title = 'Xpra', app = '' }: Props = $props();

  let iframeEl: HTMLIFrameElement;

  // Connect to Xpra HTML5 client
  // Xpra serves HTML5 at http://localhost:$PORT/index.html?session=$DISPLAY
  // We proxy this through the Go backend to avoid CORS issues
  const wsUrl = $derived(
    `/api/xpra/proxy?window=${encodeURIComponent(windowId)}`
  );

  onMount(() => {
    // Listen for xpra session events from backend
    const handleEvent = (e: MessageEvent) => {
      const msg = JSON.parse(e.data);
      if (msg.protocol !== 'xpra') return;
      // Handle window created/destroyed/focused events
      if (msg.event === 'window-created') {
        title = msg.data.title;
      }
    };

    window.addEventListener('message', handleEvent);
    return () => window.removeEventListener('message', handleEvent);
  });
</script>

<div class="xpra-tile" class:active={true}>
  <div class="xpra-header">
    <span class="xpra-title">{title}</span>
    <div class="xpra-controls">
      <button onclick={() => window.sendWs({ protocol:'xpra', method:'focus', params:{windowId} })}>↑</button>
      <button onclick={() => window.sendWs({ protocol:'xpra', method:'close', params:{windowId} })}>✕</button>
    </div>
  </div>
  <iframe
    bind:this={iframeEl}
    src="/api/xpra/html5"
    title="Xpra session"
    allow="clipboard-read; clipboard-write"
  ></iframe>
</div>

<style>
  .xpra-tile {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    background: #1a1a1a;
  }
  .xpra-header {
    display: flex;
    justify-content: space-between;
    padding: 4px 8px;
    background: #2a2a2a;
    color: #ccc;
    font-size: 12px;
  }
  iframe {
    flex: 1;
    border: none;
    width: 100%;
  }
</style>
```

### 4.2 Window Lifecycle Events

The Go backend emits WebSocket events as windows are created/destroyed by Xpra:

```json
{
  "protocol": "xpra",
  "event": "window-created",
  "data": {
    "window_id": "00001",
    "title": "Firefox",
    "geometry": [0, 0, 1024, 768]
  },
  "ts": 1234567890
}
{
  "protocol": "xpra",
  "event": "window-closed",
  "data": { "window_id": "00001" },
  "ts": 1234567891
}
{
  "protocol": "xpra",
  "event": "window-focused",
  "data": { "window_id": "00001" },
  "ts": 1234567892
}
```

---

## 5. Simplified Approach — Direct Iframe

Full Xpra WebSocket proxy is complex. A simpler integration uses Xpra's built-in HTML5 server:

### 5.1 Xpra Server Configuration

```bash
xpra start --bind=TCP --bind-web=9443 \
           --html=on \
           --start=xterm \
           --idle-timeout=0 \
           --server-Idle-Timeout=0 \
           --exit-with-children \
           :10
```

### 5.2 Backend Proxy Route

```go
// In backend main.go or ws package
import "github.com/gorilla/mux"

// Proxy /api/xpra/* to Xpra's internal HTTP server
router.PathPrefix("/api/xpra/").HandlerFunc(
    proxyHandler("http://127.0.0.1:9443", timeout(10*time.Second))
)
```

### 5.3 Frontend Iframe

```svelte
<iframe
  src="/api/xpra/index.html?session=:10"
  allow="clipboard-read; clipboard-write"
  sandbox="allow-scripts allow-same-origin allow-forms"
></iframe>
```

**Pros:** Zero additional WebSocket proxy code. Xpra's HTML5 client handles all X11 forwarding.

**Cons:** No per-window tile isolation (one iframe = all windows visible). Clipboard must be explicitly granted.

---

## 6. Launching Apps via Xpra

### 6.1 Via `apps.go` Integration

Add Xpra as an app type in the backend:

```go
// In apps.go
switch app.Type {
case "xpra":
    // Look up or create Xpra manager for this session
    mgr, ok := xpra.Managers[sessionID]
    if !ok {
        mgr = xpra.New(sessionID, 10+len(xpra.Managers))
        if err := mgr.Start(ctx); err != nil {
            return nil, err
        }
        xpra.Managers[sessionID] = mgr
    }

    cmd, err := mgr.AttachWindow(ctx, app.Command, app.Args)
    if err != nil {
        return nil, err
    }

    // Notify frontend of new window
    ws.Broadcast(ws.Message{
        Protocol: "xpra",
        Event:    "window-created",
        Data:     map[string]string{"title": app.Title},
    })
    return cmd, nil
}
```

### 6.2 App Manifest Schema

```json
{
  "type": "xpra",
  "title": "Firefox",
  "command": "firefox",
  "args": ["--new-window", "https://example.com"],
  "xpraDisplay": ":10"
}
```

---

## 7. Security Considerations

| Concern | Mitigation |
|---------|------------|
| X11 sandbox escape | Run Xpra in a nested X server (`xvfb-run`) or Docker container |
| Arbitrary app execution | Tier-1/Tier-2 security model applies (see SPEC.md §7) |
| Clipboard data exfil | Restrict clipboard access to user-approved only |
| Xpra serverDoS | Rate-limit window creation; max 20 windows per session |
| WebSocket proxy abuse | Authenticate Xpra WS connections via session token |

### Nested X Server (Recommended for MVP)

```bash
xpra start --bind-web=9443 \
           --xvfb="Xvfb :99 -screen 0 1920x1080x24" \
           --html=on \
           --start=xterm \
           :10
```

This runs Xpra inside Xvfb (virtual framebuffer), isolating it from the host X11 server.

---

## 8. Implementation Phases

### Phase 1: MVP — Single App, Direct Iframe
- [ ] Install Xpra on host
- [ ] Configure Xpra server with Xvfb on fixed display `:10`
- [ ] Go backend starts Xpra server on session init
- [ ] Backend proxies Xpra HTML5 at `/api/xpra/`
- [ ] `XpraTile.svelte` loads `/api/xpra/index.html`
- [ ] Hardcoded Firefox/terminal launch for testing

### Phase 2: Multi-Window Support
- [ ] Xpra manager tracks multiple windows per session
- [ ] Backend polls for window list (`xpra list Windows`)
- [ ] Frontend renders one iframe per window with individual headers
- [ ] Window-created/window-closed events map to tile split/unmount

### Phase 3: Dynamic App Launcher
- [ ] App manifest allows `xpra` type in `apps.go`
- [ ] User launches arbitrary GUI apps via AppLauncher
- [ ] Backend tracks window ID → app mapping

### Phase 4: Persistence & Recovery
- [ ] Session restore: reconnect to existing Xpra server
- [ ] Window state persisted (geometry, focus)
- [ ] Xpra server restart without killing apps

---

## 9. Dependencies & Installation

### Automated Setup (Recommended)

Run the setup script on the EndeavourOS host:

```bash
cd ~/.hermes/hermes-web-computer/scripts
bash setup-xpra.sh
```

This installs xpra from AUR, creates the systemd user service for auto-start,
and starts the server on display `:10` with HTML5 bridge on port `9453`.
See the script for additional flags: `bash setup-xpra.sh --help`

### Manual Install (For Reference)

```bash
# Install Xpra (EndeavourOS / Arch — AUR)
yay -S xpra

# Verify
xpra --version

# Start manual test
xpra start :10 --bind-web=9453 --html=on --start=xterm \
  --xvfb="Xvfb :10 -screen 0 1920x1080x24 -ac"

# Open browser to http://localhost:9453
```

For Go dependency:

```go
// No external Go library needed — Xpra runs as a subprocess
// import "github.com/Xpra-org/xpra-go" (if/when a Go client library exists)
```

---

## 10. Known Limitations

1. **No GPU acceleration** — Xpra over network uses encoding; latency acceptable for productivity apps, not gaming.
2. **Clipboard sync** — Requires explicit user gesture; not automatic.
3. **Audio** — Xpra supports audio forwarding but disabled by default; can enable with `--enable-audio`.
4. **Multi-window isolation** — Phase 2+ required to map individual X11 windows to tiles.
5. **Performance** — Compression settings matter; `--encoding=auto` with `--quality=50` is a good starting point.
6. **Xpra version** — HTML5 client requires Xpra v4.x+. Check `xpra --version` before deployment.

---

## 11. Reference

- [Xpra Official Site](https://xpra.org)
- [Xpra HTML5 Client Docs](https://github.com/Xpra-org/xpra/tree/master/html)
- [Xpra Protocol](https://xpra.org/trac/wiki/XpraProtocol)
- [Xpra Installation](https://xpra.org/trac/wiki/Download)