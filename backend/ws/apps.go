package ws

import (
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
			"type":         "browser",
			"browser_id":   browserSessionID,
			"url":          "about:blank",
		})})

	default:
		sess.Send(Event{Protocol: "ui", Event: "apps.error", Data: mustMarshal(map[string]string{
			"message": fmt.Sprintf("unknown app type: %s", p.Type),
		})})
	}
}
