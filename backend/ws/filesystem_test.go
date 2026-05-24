package ws

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain allows setting HERMES_HWC_ROOT before any tests run.
// Package init() runs AFTER var initializers, so we use TestMain instead.
func TestMain(m *testing.M) {
	wd, _ := os.Getwd()
	// wd is /opt/data/cache/hermes-web-computer/hermes-web-sync/backend/ws (go test ./ws/)
	// strip /ws to get /backend, strip /backend to get repo root
	repoRoot := filepath.Dir(filepath.Dir(wd))
	os.Setenv("HERMES_HWC_ROOT", repoRoot)
	os.Exit(m.Run())
}

// --- helpers ---

// drainEvents reads all pending events from the session's send channel and returns them.
func drainEvents(sess *Session) []Event {
	var events []Event
	for {
		select {
		case e := <-sess.send:
			events = append(events, e)
		default:
			return events
		}
	}
}

func jsonParams(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

type pathParam struct {
	Path string `json:"path"`
}

type writeParam struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

// --- sanitizePath tests ---

func TestSanitizePath_Root(t *testing.T) {
	got, err := sanitizePath("/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Clean(allowedRoot)
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestSanitizePath_Nested(t *testing.T) {
	got, err := sanitizePath("/backend/ws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(filepath.Clean(allowedRoot), "backend", "ws")
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestSanitizePath_Traversal(t *testing.T) {
	_, err := sanitizePath("../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

func TestSanitizePath_AbsoluteEscape(t *testing.T) {
	// Absolute paths are sandboxed: /etc/shadow becomes allowedRoot/etc/shadow
	// This is safe because the path is still under the allowed root.
	got, err := sanitizePath("/etc/shadow")
	if err != nil {
		t.Fatalf("expected sandboxed path, got error: %v", err)
	}
	if !strings.HasPrefix(got, allowedRootLazy()) {
		t.Fatalf("expected path under %q, got %q", allowedRoot, got)
	}
}

// --- handleFSList tests ---

func TestFSList_Root(t *testing.T) {
	m := newTestMultiplexer()
	sess := newTestSession()

	m.handleFSList(sess, jsonParams(pathParam{Path: "/"}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.list.response" {
		t.Fatalf("expected event 'fs.list.response', got %q (data: %s)", ev.Event, string(ev.Data))
	}

	var resp struct {
		Path    string        `json:"path"`
		Entries []fsListEntry `json:"entries"`
	}
	if err := json.Unmarshal(ev.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	names := make(map[string]bool)
	for _, e := range resp.Entries {
		names[e.Name] = true
	}

	if !names["backend"] {
		t.Error("expected 'backend' in root listing")
	}
	if !names["frontend"] {
		t.Error("expected 'frontend' in root listing")
	}
}

func TestFSList_Nested(t *testing.T) {
	m := newTestMultiplexer()
	sess := newTestSession()

	m.handleFSList(sess, jsonParams(pathParam{Path: "/backend"}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.list.response" {
		t.Fatalf("expected event 'fs.list.response', got %q", ev.Event)
	}

	var resp struct {
		Entries []fsListEntry `json:"entries"`
	}
	if err := json.Unmarshal(ev.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	names := make(map[string]bool)
	for _, e := range resp.Entries {
		names[e.Name] = true
	}

	if !names["ws"] {
		t.Error("expected 'ws' in backend listing")
	}
	if !names["cmd"] {
		t.Error("expected 'cmd' in backend listing")
	}
	if !names["layout"] {
		t.Error("expected 'layout' in backend listing")
	}
}

func TestFSList_NotFound(t *testing.T) {
	m := newTestMultiplexer()
	sess := newTestSession()

	m.handleFSList(sess, jsonParams(pathParam{Path: "/nonexistent"}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.error" {
		t.Fatalf("expected event 'fs.error', got %q", ev.Event)
	}
}

// --- handleFSRead tests ---

func TestFSRead_TextFile(t *testing.T) {
	m := newTestMultiplexer()
	sess := newTestSession()

	m.handleFSRead(sess, jsonParams(pathParam{Path: "/README.md"}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.read.response" {
		t.Fatalf("expected event 'fs.read.response', got %q", ev.Event)
	}

	var resp struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		Size     int    `json:"size"`
	}
	if err := json.Unmarshal(ev.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Encoding != "utf8" {
		t.Errorf("expected encoding 'utf8', got %q", resp.Encoding)
	}
	if resp.Size == 0 {
		t.Error("expected non-zero file size")
	}
}

func TestFSRead_NotFound(t *testing.T) {
	m := newTestMultiplexer()
	sess := newTestSession()

	m.handleFSRead(sess, jsonParams(pathParam{Path: "/nope.txt"}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.error" {
		t.Fatalf("expected event 'fs.error', got %q", ev.Event)
	}
}

// --- handleFSWrite tests ---

func TestFSWrite_Create(t *testing.T) {
	tmpDir := t.TempDir()

	oldRoot := allowedRoot
	allowedRoot = tmpDir
	defer func() { allowedRoot = oldRoot }()

	m := newTestMultiplexer()
	sess := newTestSession()
	testFile := "/test_create.txt"
	testContent := "hello filesystem write"

	m.handleFSWrite(sess, jsonParams(writeParam{
		Path:    testFile,
		Content: testContent,
	}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.write.response" {
		t.Fatalf("expected event 'fs.write.response', got %q", ev.Event)
	}

	var resp struct {
		Path         string `json:"path"`
		BytesWritten int    `json:"bytes_written"`
	}
	if err := json.Unmarshal(ev.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.BytesWritten != len(testContent) {
		t.Errorf("expected %d bytes written, got %d", len(testContent), resp.BytesWritten)
	}

	// Verify file content on disk
	got, err := os.ReadFile(filepath.Join(tmpDir, "test_create.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(got) != testContent {
		t.Errorf("expected content %q, got %q", testContent, string(got))
	}
}

func TestFSWrite_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()

	oldRoot := allowedRoot
	allowedRoot = tmpDir
	defer func() { allowedRoot = oldRoot }()

	// Create initial file
	initialPath := filepath.Join(tmpDir, "overwrite_test.txt")
	if err := os.WriteFile(initialPath, []byte("original"), 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	m := newTestMultiplexer()
	sess := newTestSession()
	newContent := "overwritten content"

	m.handleFSWrite(sess, jsonParams(writeParam{
		Path:    "/overwrite_test.txt",
		Content: newContent,
	}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.write.response" {
		t.Fatalf("expected event 'fs.write.response', got %q", ev.Event)
	}

	// Verify overwritten content
	got, err := os.ReadFile(initialPath)
	if err != nil {
		t.Fatalf("failed to read overwritten file: %v", err)
	}
	if string(got) != newContent {
		t.Errorf("expected content %q, got %q", newContent, string(got))
	}
}

// --- handleFSStat tests ---

func TestFSStat_File(t *testing.T) {
	m := newTestMultiplexer()
	sess := newTestSession()

	m.handleFSStat(sess, jsonParams(pathParam{Path: "/README.md"}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.stat.response" {
		t.Fatalf("expected event 'fs.stat.response', got %q", ev.Event)
	}

	var resp struct {
		Path   string `json:"path"`
		Exists bool   `json:"exists"`
		Type   string `json:"type"`
	}
	if err := json.Unmarshal(ev.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Exists {
		t.Error("expected exists=true")
	}
	if resp.Type != "file" {
		t.Errorf("expected type 'file', got %q", resp.Type)
	}
}

func TestFSStat_Dir(t *testing.T) {
	m := newTestMultiplexer()
	sess := newTestSession()

	m.handleFSStat(sess, jsonParams(pathParam{Path: "/backend"}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.stat.response" {
		t.Fatalf("expected event 'fs.stat.response', got %q", ev.Event)
	}

	var resp struct {
		Path   string `json:"path"`
		Exists bool   `json:"exists"`
		Type   string `json:"type"`
	}
	if err := json.Unmarshal(ev.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Exists {
		t.Error("expected exists=true")
	}
	if resp.Type != "dir" {
		t.Errorf("expected type 'dir', got %q", resp.Type)
	}
}

func TestFSStat_NotFound(t *testing.T) {
	m := newTestMultiplexer()
	sess := newTestSession()

	m.handleFSStat(sess, jsonParams(pathParam{Path: "/this_path_does_not_exist_12345"}))

	events := drainEvents(sess)
	if len(events) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	ev := events[len(events)-1]
	if ev.Event != "fs.stat.response" {
		t.Fatalf("expected event 'fs.stat.response', got %q", ev.Event)
	}

	var resp struct {
		Exists bool `json:"exists"`
	}
	if err := json.Unmarshal(ev.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Exists {
		t.Error("expected exists=false for non-existent path")
	}
}
