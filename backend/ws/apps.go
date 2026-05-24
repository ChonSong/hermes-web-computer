package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// AppType describes a launchable application.
type AppType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// handleAppsList returns available app types.
func (m *Multiplexer) handleAppsList(sess *Session) {
	apps := []AppType{
		{ID: "terminal", Name: "Terminal", Icon: "⬛"},
		{ID: "editor", Name: "Editor", Icon: "📝"},
		{ID: "preview", Name: "Preview", Icon: "👁"},
		{ID: "browser", Name: "Browser", Icon: "🌐"},
		{ID: "xpra", Name: "Xpra", Icon: "🪟"},
	}
	sess.Send(Event{Protocol: "ui", Event: "apps.list.response", Data: mustMarshal(map[string]interface{}{"apps": apps})})
}

// handleAppsLaunch launches a new app instance.
func (m *Multiplexer) handleAppsLaunch(sess *Session, params json.RawMessage) {
	var p struct {
		Type string `json:"type"`
		Path string `json:"path,omitempty"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "apps.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}

	switch p.Type {
	case "terminal":
		ptyID := fmt.Sprintf("pty_%d", time.Now().UnixNano())
		cmd := exec.Command("bash", "-i")
		cmd.Env = append(os.Environ(), "TERM=xterm-256color")
		ptySession, err := m.supervisor.Start(ptyID, cmd)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "apps.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
			return
		}
		go m.forwardPTYOutput(sess, ptySession)
		sess.Send(Event{Protocol: "ui", Event: "apps.launch.response", Data: mustMarshal(map[string]interface{}{
			"type":   "terminal",
			"pty_id": ptyID,
		})})

	case "editor":
		sess.Send(Event{Protocol: "ui", Event: "apps.launch.response", Data: mustMarshal(map[string]interface{}{
			"type": "editor",
			"path": p.Path,
		})})

	case "preview":
		sess.Send(Event{Protocol: "ui", Event: "apps.launch.response", Data: mustMarshal(map[string]interface{}{
			"type": "preview",
		})})

	case "browser":
		browserSessionID := fmt.Sprintf("browser_%d", time.Now().UnixNano())
		if _, err := m.browser.Launch(browserSessionID); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "apps.error", Data: mustMarshal(map[string]string{"message": err.Error()})})
			return
		}
		sess.Send(Event{Protocol: "ui", Event: "apps.launch.response", Data: mustMarshal(map[string]interface{}{
			"type":       "browser",
			"browser_id": browserSessionID,
			"url":        "about:blank",
		})})

	case "file-manager", "agent", "dashboard", "audio":
		// These are panel-based features, not tile-based apps
		// Return success so the dock indicator updates, but no new tile is created
		sess.Send(Event{Protocol: "ui", Event: "apps.launch.response", Data: mustMarshal(map[string]interface{}{
			"type": p.Type,
			"note": "panel feature - no tile launched",
		})})

	case "xpra":
		// XPra escape hatch — start the Xpra server and report the session URL
		m.mu.RLock()
		xm := m.xpraMgr
		m.mu.RUnlock()

		if xm == nil {
			sess.Send(Event{Protocol: "ui", Event: "apps.error", Data: mustMarshal(map[string]string{
				"message": "xpra not initialized (HERMES_XPRA_DISPLAY not set or xpra not installed)",
			})})
			break
		}

		if !xm.IsRunning() {
			// Try to start on first use
			ctx := context.Background()
			if err := xm.Start(ctx); err != nil {
				sess.Send(Event{Protocol: "ui", Event: "apps.error", Data: mustMarshal(map[string]string{
					"message": fmt.Sprintf("xpra start failed: %v", err),
				})})
				break
			}
		}

		sess.Send(Event{Protocol: "ui", Event: "apps.launch.response", Data: mustMarshal(map[string]interface{}{
			"type":      "xpra",
			"http_url":  xm.HTTPURL(),
			"display":   xm.Display(),
			"tile_type": "xpra",
		})})

	default:
		sess.Send(Event{Protocol: "ui", Event: "apps.error", Data: mustMarshal(map[string]string{
			"message": fmt.Sprintf("unknown app type: %s", p.Type),
		})})
	}
}
