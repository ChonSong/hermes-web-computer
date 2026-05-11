package ws

import (
	"encoding/json"
	"testing"
)

func TestContextManager_SetFocus(t *testing.T) {
	cm := NewContextManager()
	sess := newTestSession()

	// Set focus on a terminal tile
	ctx1 := &TileContext{
		TileID:      "tile_terminal",
		ContentType: "xterm",
		PTYID:       "pty_123",
	}
	cm.SetFocus(ctx1, sess)

	// Verify focus is set
	got := cm.GetFocus()
	if got == nil {
		t.Fatal("expected focus context, got nil")
	}
	if got.TileID != "tile_terminal" {
		t.Errorf("expected tile_id 'tile_terminal', got %q", got.TileID)
	}
	if got.ContentType != "xterm" {
		t.Errorf("expected content_type 'xterm', got %q", got.ContentType)
	}

	// Verify events were emitted
	events := captureEvents(sess)
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (ui.focus.changed + agent.context), got %d", len(events))
	}

	// Check ui.focus.changed event
	foundFocusChanged := false
	foundAgentContext := false
	for _, e := range events {
		if e.Event == "ui.focus.changed" {
			foundFocusChanged = true
			var data map[string]interface{}
			if err := json.Unmarshal(e.Data, &data); err != nil {
				t.Fatalf("failed to parse ui.focus.changed data: %v", err)
			}
			if data["tile_id"] != "tile_terminal" {
				t.Errorf("expected tile_id 'tile_terminal' in event, got %v", data["tile_id"])
			}
			if data["content_type"] != "xterm" {
				t.Errorf("expected content_type 'xterm' in event, got %v", data["content_type"])
			}
		}
		if e.Event == "agent.context" {
			foundAgentContext = true
			var data map[string]interface{}
			if err := json.Unmarshal(e.Data, &data); err != nil {
				t.Fatalf("failed to parse agent.context data: %v", err)
			}
			if data["scope"] == nil {
				t.Error("agent.context missing 'scope' field")
			}
		}
	}
	if !foundFocusChanged {
		t.Error("expected ui.focus.changed event not found")
	}
	if !foundAgentContext {
		t.Error("expected agent.context event not found")
	}
}

func TestContextManager_FocusChange(t *testing.T) {
	cm := NewContextManager()
	sess := newTestSession()

	// Set initial focus
	ctx1 := &TileContext{TileID: "tile_1", ContentType: "xterm", PTYID: "pty_1"}
	cm.SetFocus(ctx1, sess)

	// Clear captured events
	captureEvents(sess)

	// Change focus to a different tile
	ctx2 := &TileContext{TileID: "tile_2", ContentType: "monaco", Path: "/file.go"}
	cm.SetFocus(ctx2, sess)

	events := captureEvents(sess)
	foundFocusChanged := false
	for _, e := range events {
		if e.Event == "ui.focus.changed" {
			foundFocusChanged = true
			var data map[string]interface{}
			if err := json.Unmarshal(e.Data, &data); err != nil {
				t.Fatalf("failed to parse ui.focus.changed data: %v", err)
			}
			if data["tile_id"] != "tile_2" {
				t.Errorf("expected new tile_id 'tile_2', got %v", data["tile_id"])
			}
			if data["previous_tile_id"] != "tile_1" {
				t.Errorf("expected previous_tile_id 'tile_1', got %v", data["previous_tile_id"])
			}
		}
	}
	if !foundFocusChanged {
		t.Error("expected ui.focus.changed event after focus change")
	}
}

func TestContextManager_BuildAgentScope(t *testing.T) {
	tests := []struct {
		name     string
		context  *TileContext
		wantPart string
	}{
		{
			name:     "terminal with PTY",
			context:  &TileContext{TileID: "t1", ContentType: "xterm", PTYID: "pty_1"},
			wantPart: "Focused tile: terminal (pty_id=pty_1)",
		},
		{
			name:     "editor with path",
			context:  &TileContext{TileID: "t2", ContentType: "monaco", Path: "/src/main.go"},
			wantPart: "Focused tile: editor (path=/src/main.go)",
		},
		{
			name:     "browser with session",
			context:  &TileContext{TileID: "t3", ContentType: "browser", BrowserID: "br_1"},
			wantPart: "Focused tile: browser (browser_id=br_1)",
		},
		{
			name:     "preview",
			context:  &TileContext{TileID: "t4", ContentType: "preview"},
			wantPart: "Focused tile: preview",
		},
		{
			name:     "welcome",
			context:  &TileContext{TileID: "t5", ContentType: "welcome"},
			wantPart: "Focused tile: welcome",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewContextManager()
			cm.SetFocus(tt.context, nil)
			scope := cm.BuildAgentScope()
			if scope != tt.wantPart {
				t.Errorf("got scope %q, want %q", scope, tt.wantPart)
			}
		})
	}
}

func TestContextManager_NoFocus(t *testing.T) {
	cm := NewContextManager()

	// No focus set
	got := cm.GetFocus()
	if got != nil {
		t.Errorf("expected nil focus, got %+v", got)
	}

	scope := cm.BuildAgentScope()
	if scope != "" {
		t.Errorf("expected empty scope with no focus, got %q", scope)
	}
}

func TestHandleFocusChange(t *testing.T) {
	mux := newTestMultiplexer()
	sess := newTestSession()

	params := json.RawMessage(`{"tile_id":"tile_term_1","content_type":"xterm","pty_id":"pty_999"}`)
	mux.handleFocusChange(sess, params, "test_session")

	events := captureEvents(sess)
	foundFocusChanged := false
	for _, e := range events {
		if e.Event == "ui.focus.changed" {
			foundFocusChanged = true
			var data map[string]interface{}
			if err := json.Unmarshal(e.Data, &data); err != nil {
				t.Fatalf("failed to parse data: %v", err)
			}
			if data["tile_id"] != "tile_term_1" {
				t.Errorf("expected tile_id 'tile_term_1', got %v", data["tile_id"])
			}
		}
	}
	if !foundFocusChanged {
		t.Error("expected ui.focus.changed event")
	}

	// Verify context manager has the focus
	ctx := mux.contextMgr.GetFocus()
	if ctx == nil {
		t.Fatal("expected context manager to have focus set")
	}
	if ctx.TileID != "tile_term_1" {
		t.Errorf("expected tile_id 'tile_term_1', got %q", ctx.TileID)
	}
}

func TestHandleFocusChange_MissingFields(t *testing.T) {
	mux := newTestMultiplexer()
	sess := newTestSession()

	// Missing content_type
	params := json.RawMessage(`{"tile_id":"tile_1"}`)
	mux.handleFocusChange(sess, params, "test_session")

	events := captureEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected error event")
	}

	foundError := false
	for _, e := range events {
		if e.Event == "ui.focus.error" {
			foundError = true
		}
	}
	if !foundError {
		t.Error("expected ui.focus.error event for missing content_type")
	}
}

func TestHandleFocusChange_InvalidJSON(t *testing.T) {
	mux := newTestMultiplexer()
	sess := newTestSession()

	params := json.RawMessage(`{bad json}`)
	mux.handleFocusChange(sess, params, "test_session")

	events := captureEvents(sess)
	foundError := false
	for _, e := range events {
		if e.Event == "ui.focus.error" {
			foundError = true
		}
	}
	if !foundError {
		t.Error("expected ui.focus.error event for invalid JSON")
	}
}

func TestContextManager_BrowserFocus(t *testing.T) {
	cm := NewContextManager()
	sess := newTestSession()

	ctx := &TileContext{
		TileID:      "tile_browser",
		ContentType: "browser",
		BrowserID:   "browser_abc",
	}
	cm.SetFocus(ctx, sess)

	events := captureEvents(sess)
	for _, e := range events {
		if e.Event == "ui.focus.changed" {
			var data map[string]interface{}
			if err := json.Unmarshal(e.Data, &data); err != nil {
				t.Fatalf("failed to parse data: %v", err)
			}
			if data["browser_id"] != "browser_abc" {
				t.Errorf("expected browser_id 'browser_abc', got %v", data["browser_id"])
			}
		}
	}
}

func TestHandleFocusChange_EditorWithPath(t *testing.T) {
	mux := newTestMultiplexer()
	sess := newTestSession()

	params := json.RawMessage(`{"tile_id":"tile_editor","content_type":"monaco","path":"/src/main.go"}`)
	mux.handleFocusChange(sess, params, "test_session")

	ctx := mux.contextMgr.GetFocus()
	if ctx == nil {
		t.Fatal("expected focus context")
	}
	if ctx.Path != "/src/main.go" {
		t.Errorf("expected path '/src/main.go', got %q", ctx.Path)
	}

	scope := mux.contextMgr.BuildAgentScope()
	if scope == "" {
		t.Error("expected non-empty scope for editor")
	}
}
