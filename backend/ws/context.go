package ws

import (
	"encoding/json"
	"fmt"
	"sync"

	"hermes-web-computer/backend/telemetry"
)

// TileContext describes the active context of a focused tile.
type TileContext struct {
	TileID      string `json:"tile_id"`
	ContentType string `json:"content_type"` // "xterm", "monaco", "browser", "preview", "welcome"
	PTYID       string `json:"pty_id,omitempty"`
	BrowserID   string `json:"browser_id,omitempty"`
	Path        string `json:"path,omitempty"`
}

// ContextManager tracks the currently focused tile and provides
// auto-scoped context for agent responses.
type ContextManager struct {
	mu      sync.RWMutex
	current *TileContext
}

// NewContextManager creates a new context manager with no focused tile.
func NewContextManager() *ContextManager {
	return &ContextManager{}
}

// SetFocus updates the currently focused tile and returns the previous context.
// Emits an agent.context event on the session if the focus actually changed.
func (cm *ContextManager) SetFocus(ctx *TileContext, sess *Session) *TileContext {
	cm.mu.Lock()
	prev := cm.current
	cm.current = ctx
	cm.mu.Unlock()

	if sess != nil {
		cm.emitFocusEvent(sess, ctx, prev)
	}
	return prev
}

// GetFocus returns the currently focused tile context.
func (cm *ContextManager) GetFocus() *TileContext {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	if cm.current == nil {
		return nil
	}
	cpy := *cm.current
	return &cpy
}

// BuildAgentScope returns a system-scoping hint string for the agent,
// describing which tile is active so the agent can auto-scope responses.
func (cm *ContextManager) BuildAgentScope() string {
	ctx := cm.GetFocus()
	if ctx == nil {
		return ""
	}

	switch ctx.ContentType {
	case "xterm":
		if ctx.PTYID != "" {
			return fmt.Sprintf("Focused tile: terminal (pty_id=%s)", ctx.PTYID)
		}
		return "Focused tile: terminal"
	case "monaco":
		if ctx.Path != "" {
			return fmt.Sprintf("Focused tile: editor (path=%s)", ctx.Path)
		}
		return "Focused tile: editor"
	case "browser":
		if ctx.BrowserID != "" {
			return fmt.Sprintf("Focused tile: browser (browser_id=%s)", ctx.BrowserID)
		}
		return "Focused tile: browser"
	case "preview":
		return "Focused tile: preview"
	case "welcome":
		return "Focused tile: welcome"
	default:
		return fmt.Sprintf("Focused tile: %s (tile_id=%s)", ctx.ContentType, ctx.TileID)
	}
}

// emitFocusEvent sends a ui.focus.changed event to the client and an
// agent.context event for the agent to receive updated context.
func (cm *ContextManager) emitFocusEvent(sess *Session, current, previous *TileContext) {
	if current == nil {
		return
	}

	data := map[string]interface{}{
		"tile_id":      current.TileID,
		"content_type": current.ContentType,
	}
	if current.PTYID != "" {
		data["pty_id"] = current.PTYID
	}
	if current.BrowserID != "" {
		data["browser_id"] = current.BrowserID
	}
	if current.Path != "" {
		data["path"] = current.Path
	}
	if previous != nil {
		data["previous_tile_id"] = previous.TileID
	}

	sess.Send(Event{
		Protocol: "ui",
		Event:    "ui.focus.changed",
		Data:     json.RawMessage(mustMarshal(data)),
	})

	// Also emit agent context so the agent can auto-scope
	sess.Send(Event{
		Protocol: "agent",
		Event:    "agent.context",
		Data:     json.RawMessage(mustMarshal(map[string]interface{}{
			"focus":  current,
			"scope":  cm.BuildAgentScope(),
		})),
	})
}

// handleFocusChange processes a ui.focus.change request from the UI.
func (m *Multiplexer) handleFocusChange(sess *Session, params json.RawMessage, sessionID string) {
	var p struct {
		TileID      string `json:"tile_id"`
		ContentType string `json:"content_type"`
		PTYID       string `json:"pty_id,omitempty"`
		BrowserID   string `json:"browser_id,omitempty"`
		Path        string `json:"path,omitempty"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		sess.Send(Event{Protocol: "ui", Event: "ui.focus.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
		return
	}

	if p.TileID == "" || p.ContentType == "" {
		sess.Send(Event{Protocol: "ui", Event: "ui.focus.error", Data: json.RawMessage(`{"message":"tile_id and content_type are required"}`)})
		return
	}

	// Also update layout tree focus tracking
	m.layout.SetFocus(p.TileID)

	tileCtx := &TileContext{
		TileID:      p.TileID,
		ContentType: p.ContentType,
		PTYID:       p.PTYID,
		BrowserID:   p.BrowserID,
		Path:        p.Path,
	}

	m.contextMgr.SetFocus(tileCtx, sess)

	// Telemetry
	if m.telemetry != nil {
		m.telemetry.Write(telemetry.Event{
			SessionID: sessionID,
			Type:      "ui.focus.change",
			Command:   fmt.Sprintf("%s:%s", p.ContentType, p.TileID),
		})
	}
}
