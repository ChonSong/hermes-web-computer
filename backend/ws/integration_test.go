package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

func readEventWithTimeout(t *testing.T, ctx context.Context, conn *websocket.Conn, timeout time.Duration) (Event, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, msg, err := conn.Read(ctx)
	if err != nil {
		return Event{}, fmt.Errorf("read error: %w", err)
	}

	var event Event
	if err := json.Unmarshal(msg, &event); err != nil {
		return Event{}, fmt.Errorf("unmarshal error: %w", err)
	}
	return event, nil
}

func readEventsUntil(t *testing.T, ctx context.Context, conn *websocket.Conn, timeout time.Duration, match func(Event) bool) (Event, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		event, err := readEventWithTimeout(t, ctx, conn, remaining)
		if err != nil {
			return Event{}, err
		}
		if match(event) {
			return event, nil
		}
	}
	return Event{}, fmt.Errorf("timeout waiting for matching event")
}

func sendEnvelope(t *testing.T, ctx context.Context, conn *websocket.Conn, env Envelope) error {
	t.Helper()
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

func setupTestServer(t *testing.T) (*httptest.Server, *Multiplexer) {
	t.Helper()
	m := NewMultiplexer()
	server := httptest.NewServer(m.Router())
	t.Cleanup(server.Close)
	return server, m
}

func connectWS(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	t.Cleanup(func() { conn.Close(websocket.StatusNormalClosure, "test done") })
	return conn
}

// TestWSConnect verifies that a client can connect to /ws and receive the
// layout.initial event immediately after connection.
func TestWSConnect(t *testing.T) {
	server, _ := setupTestServer(t)
	conn := connectWS(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	event, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "layout.initial"
	})
	if err != nil {
		t.Fatalf("did not receive layout.initial event: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		t.Fatalf("failed to parse layout.initial data: %v", err)
	}

	if data["layout_version"] == nil {
		t.Error("layout.initial missing layout_version")
	}
	tree, ok := data["tree"].(map[string]interface{})
	if !ok {
		t.Fatal("layout.initial missing tree object")
	}
	if tree["pty_id"] == nil {
		t.Error("layout.initial missing pty_id in tree")
	}
}

// TestWSFSRoundTrip verifies sending an fs.list request and receiving the
// corresponding fs.list.response with directory entries.
func TestWSFSRoundTrip(t *testing.T) {
	server, _ := setupTestServer(t)
	conn := connectWS(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Wait for layout.initial
	_, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "layout.initial"
	})
	if err != nil {
		t.Fatalf("did not receive layout.initial: %v", err)
	}

	// Send fs.list request
	params := mustMarshal(map[string]string{"path": "/"})
	err = sendEnvelope(t, ctx, conn, Envelope{
		Protocol: "ui",
		Method:   "fs.list",
		Params:   params,
		ID:       "fs-1",
		Ts:       time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("failed to send fs.list: %v", err)
	}

	// Wait for fs.list.response
	event, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "fs.list.response"
	})
	if err != nil {
		t.Fatalf("did not receive fs.list.response: %v", err)
	}

	var data struct {
		Path    string        `json:"path"`
		Entries []fsListEntry `json:"entries"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		t.Fatalf("failed to parse fs.list.response data: %v", err)
	}

	if data.Path == "" {
		t.Error("fs.list.response missing path")
	}
	if len(data.Entries) == 0 {
		t.Error("fs.list.response has no entries")
	}

	for _, entry := range data.Entries {
		if entry.Name == "" {
			t.Error("fs list entry has empty name")
		}
		if entry.Type != "file" && entry.Type != "dir" && entry.Type != "symlink" {
			t.Errorf("fs list entry %q has invalid type: %s", entry.Name, entry.Type)
		}
	}
}

// TestWSFSWriteReadRoundTrip verifies creating a file via fs.write and then
// reading it back via fs.read to confirm the content matches.
func TestWSFSWriteReadRoundTrip(t *testing.T) {
	server, _ := setupTestServer(t)
	conn := connectWS(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Wait for layout.initial
	_, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "layout.initial"
	})
	if err != nil {
		t.Fatalf("did not receive layout.initial: %v", err)
	}

	testContent := "hello from ws integration test " + fmt.Sprintf("%d", time.Now().UnixNano())
	testPath := filepath.Join("/opt/data/hermes-web-computer", "test_ws_roundtrip.txt")

	// Write the file
	writeParams := mustMarshal(map[string]string{
		"path":    testPath,
		"content": testContent,
	})
	err = sendEnvelope(t, ctx, conn, Envelope{
		Protocol: "ui",
		Method:   "fs.write",
		Params:   writeParams,
		ID:       "write-1",
		Ts:       time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("failed to send fs.write: %v", err)
	}

	_, err = readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "fs.write.response"
	})
	if err != nil {
		t.Fatalf("did not receive fs.write.response: %v", err)
	}

	// Read the file back
	readParams := mustMarshal(map[string]string{"path": testPath})
	err = sendEnvelope(t, ctx, conn, Envelope{
		Protocol: "ui",
		Method:   "fs.read",
		Params:   readParams,
		ID:       "read-1",
		Ts:       time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("failed to send fs.read: %v", err)
	}

	event, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "fs.read.response"
	})
	if err != nil {
		t.Fatalf("did not receive fs.read.response: %v", err)
	}

	var readData struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Size    int    `json:"size"`
	}
	if err := json.Unmarshal(event.Data, &readData); err != nil {
		t.Fatalf("failed to parse fs.read.response data: %v", err)
	}

	if readData.Content != testContent {
		t.Errorf("content mismatch: got %q, want %q", readData.Content, testContent)
	}
	if readData.Size != len(testContent) {
		t.Errorf("size mismatch: got %d, want %d", readData.Size, len(testContent))
	}

	os.Remove(testPath)
}

// TestWSAppLaunchAndPTY verifies launching a terminal app and receiving a
// response with a pty_id.
func TestWSAppLaunchAndPTY(t *testing.T) {
	server, _ := setupTestServer(t)
	conn := connectWS(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "layout.initial"
	})
	if err != nil {
		t.Fatalf("did not receive layout.initial: %v", err)
	}

	launchParams := mustMarshal(map[string]string{"type": "terminal"})
	err = sendEnvelope(t, ctx, conn, Envelope{
		Protocol: "ui",
		Method:   "apps.launch",
		Params:   launchParams,
		ID:       "launch-1",
		Ts:       time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("failed to send apps.launch: %v", err)
	}

	event, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "apps.launch.response"
	})
	if err != nil {
		t.Fatalf("did not receive apps.launch.response: %v", err)
	}

	var data struct {
		Type  string `json:"type"`
		PTYID string `json:"pty_id"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		t.Fatalf("failed to parse apps.launch.response data: %v", err)
	}

	if data.Type != "terminal" {
		t.Errorf("expected type 'terminal', got %q", data.Type)
	}
	if data.PTYID == "" {
		t.Error("apps.launch.response missing pty_id")
	}
	if !strings.HasPrefix(data.PTYID, "pty_") {
		t.Errorf("pty_id does not have expected prefix: %q", data.PTYID)
	}
}

// TestWSMultiProtocol verifies sequential messages for different protocols
// are routed correctly.
func TestWSMultiProtocol(t *testing.T) {
	server, _ := setupTestServer(t)
	conn := connectWS(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "layout.initial"
	})
	if err != nil {
		t.Fatalf("did not receive layout.initial: %v", err)
	}

	// Send fs.stat, wait for response
	fsStatParams := mustMarshal(map[string]string{"path": "/"})
	err = sendEnvelope(t, ctx, conn, Envelope{
		Protocol: "ui", Method: "fs.stat", Params: fsStatParams,
		ID: "stat-1", Ts: time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("failed to send fs.stat: %v", err)
	}

	event, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "fs.stat.response"
	})
	if err != nil {
		t.Fatalf("did not receive fs.stat.response: %v", err)
	}

	var statData map[string]interface{}
	if err := json.Unmarshal(event.Data, &statData); err != nil {
		t.Fatalf("failed to parse fs.stat.response: %v", err)
	}
	if statData["exists"] != true {
		t.Error("fs.stat.response should report exists=true for root")
	}

	// Send apps.list, wait for response
	err = sendEnvelope(t, ctx, conn, Envelope{
		Protocol: "ui", Method: "apps.list",
		ID: "apps-1", Ts: time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("failed to send apps.list: %v", err)
	}

	event, err = readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "apps.list.response"
	})
	if err != nil {
		t.Fatalf("did not receive apps.list.response: %v", err)
	}

	var appsData map[string]interface{}
	if err := json.Unmarshal(event.Data, &appsData); err != nil {
		t.Fatalf("failed to parse apps.list.response: %v", err)
	}
	if appsData["apps"] == nil {
		t.Error("apps.list.response missing apps array")
	}
}

// TestWSReconnect verifies that a client can disconnect and reconnect,
// receiving a fresh layout.initial event on the new session.
func TestWSReconnect(t *testing.T) {
	server, _ := setupTestServer(t)

	// First connection
	ctx1, cancel1 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel1()
	conn1 := connectWS(t, server)
	_, err := readEventsUntil(t, ctx1, conn1, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "layout.initial"
	})
	if err != nil {
		t.Fatalf("first connection did not receive layout.initial: %v", err)
	}
	conn1.Close(websocket.StatusNormalClosure, "reconnect test")
	time.Sleep(500 * time.Millisecond)

	// Reconnect
	ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()
	conn2 := connectWS(t, server)
	event, err := readEventsUntil(t, ctx2, conn2, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "layout.initial"
	})
	if err != nil {
		t.Fatalf("second connection did not receive layout.initial: %v", err)
	}

	// Verify new session works
	params := mustMarshal(map[string]string{"path": "/"})
	err = sendEnvelope(t, ctx2, conn2, Envelope{
		Protocol: "ui", Method: "fs.list", Params: params,
		ID: "fs-reconnect", Ts: time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("failed to send fs.list on reconnected session: %v", err)
	}

	_, err = readEventsUntil(t, ctx2, conn2, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "fs.list.response"
	})
	if err != nil {
		t.Fatalf("reconnected session did not receive fs.list.response: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		t.Fatalf("failed to parse layout.initial data: %v", err)
	}
	tree, ok := data["tree"].(map[string]interface{})
	if !ok || tree["pty_id"] == nil {
		t.Error("reconnected session missing pty_id in layout.initial tree")
	}
}

// TestWSErrorHandling verifies that the server handles error conditions gracefully.
func TestWSErrorHandling(t *testing.T) {
	server, _ := setupTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn := connectWS(t, server)

	_, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "layout.initial"
	})
	if err != nil {
		t.Fatalf("did not receive layout.initial: %v", err)
	}

	// Send malformed JSON — connection should survive
	err = conn.Write(ctx, websocket.MessageText, []byte(`{bad json!!!`))
	if err != nil {
		t.Fatalf("failed to send malformed JSON: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Send a valid request after the malformed JSON
	params := mustMarshal(map[string]string{"path": "/"})
	err = sendEnvelope(t, ctx, conn, Envelope{
		Protocol: "ui", Method: "fs.list", Params: params,
		ID: "after-bad-json", Ts: time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("failed to send fs.list after malformed JSON: %v", err)
	}

	event, err := readEventsUntil(t, ctx, conn, 15*time.Second, func(e Event) bool {
		return e.Protocol == "ui" && e.Event == "fs.list.response"
	})
	if err != nil {
		t.Fatalf("connection broken after malformed JSON, did not receive fs.list.response: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		t.Fatalf("failed to parse fs.list.response: %v", err)
	}
	if data["entries"] == nil {
		t.Error("fs.list.response missing entries")
	}
}
