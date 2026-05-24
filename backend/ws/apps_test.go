package ws

import (
	"encoding/json"
	"testing"

	"hermes-web-computer/backend/layout"
	"hermes-web-computer/backend/pty"
	"hermes-web-computer/backend/security"
)

func newTestMultiplexer() *Multiplexer {
	m := &Multiplexer{
		sessions:   make(map[string]*Session),
		supervisor: pty.NewSupervisor(),
		enforcer:   security.NewEnforcer(),
		layout:     layout.NewRoot("xterm"),
		contextMgr: NewContextManager(),
	}
	m.enforcer.UseDefaults()
	return m
}

func newTestSession() *Session {
	return &Session{
		send: make(chan Event, 64),
		done: make(chan struct{}),
	}
}

// captureEvents drains all pending events from a real Session's send channel.
func captureEvents(sess *Session) []Event {
	var out []Event
	for {
		select {
		case e := <-sess.send:
			out = append(out, e)
		default:
			return out
		}
	}
}

func lastEvent(sess *Session) Event {
	events := captureEvents(sess)
	if len(events) == 0 {
		return Event{}
	}
	return events[len(events)-1]
}

func TestAppsList(t *testing.T) {
	mux := newTestMultiplexer()
	sess := newTestSession()

	mux.handleAppsList(sess)

	e := lastEvent(sess)
	if e.Event != "apps.list.response" {
		t.Fatalf("expected event apps.list.response, got %s", e.Event)
	}

	var result struct {
		Apps []AppType `json:"apps"`
	}
	if err := json.Unmarshal(e.Data, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(result.Apps) != 5 {
		t.Fatalf("expected 5 app types, got %d", len(result.Apps))
	}

	expected := map[string]string{
		"terminal": "Terminal",
		"editor":   "Editor",
		"preview":  "Preview",
		"browser":  "Browser",
		"xpra":     "Xpra",
	}
	for _, app := range result.Apps {
		wantName, ok := expected[app.ID]
		if !ok {
			t.Errorf("unexpected app ID: %s", app.ID)
			continue
		}
		if app.Name != wantName {
			t.Errorf("app %s: expected name %q, got %q", app.ID, wantName, app.Name)
		}
	}
}

func TestAppsLaunch_Terminal(t *testing.T) {
	mux := newTestMultiplexer()
	sess := newTestSession()

	params := json.RawMessage(`{"type":"terminal"}`)
	mux.handleAppsLaunch(sess, params)

	e := lastEvent(sess)
	if e.Event != "apps.launch.response" {
		t.Fatalf("expected event apps.launch.response, got %s", e.Event)
	}

	var result struct {
		Type  string `json:"type"`
		PtyID string `json:"pty_id"`
	}
	if err := json.Unmarshal(e.Data, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Type != "terminal" {
		t.Errorf("expected type terminal, got %s", result.Type)
	}
	if result.PtyID == "" {
		t.Error("expected non-empty pty_id")
	}

	// Verify the PTY session was actually created in the supervisor.
	if !mux.supervisor.Exists(result.PtyID) {
		t.Errorf("PTY session %s not found in supervisor", result.PtyID)
	}
}

func TestAppsLaunch_Editor(t *testing.T) {
	mux := newTestMultiplexer()
	sess := newTestSession()

	params := json.RawMessage(`{"type":"editor","path":"/file.go"}`)
	mux.handleAppsLaunch(sess, params)

	e := lastEvent(sess)
	if e.Event != "apps.launch.response" {
		t.Fatalf("expected event apps.launch.response, got %s", e.Event)
	}

	var result struct {
		Type string `json:"type"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(e.Data, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Type != "editor" {
		t.Errorf("expected type editor, got %s", result.Type)
	}
	if result.Path != "/file.go" {
		t.Errorf("expected path /file.go, got %s", result.Path)
	}
}

func TestAppsLaunch_Invalid(t *testing.T) {
	mux := newTestMultiplexer()
	sess := newTestSession()

	params := json.RawMessage(`{"type":"bogus"}`)
	mux.handleAppsLaunch(sess, params)

	e := lastEvent(sess)
	if e.Event != "apps.error" {
		t.Fatalf("expected event apps.error, got %s", e.Event)
	}

	var result struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(e.Data, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Message == "" {
		t.Error("expected non-empty error message")
	}
}
