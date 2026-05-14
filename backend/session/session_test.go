package session

import (
	"testing"
)

func TestSessionStore(t *testing.T) {
	tmp := t.TempDir()

	store, err := NewStore(tmp)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Create a session
	sess, err := store.New("/tmp/workspace", "test-model")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if sess.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if sess.Title != "New conversation" {
		t.Errorf("expected title 'New conversation', got %q", sess.Title)
	}
	t.Logf("created session id=%s", sess.ID)

	// List all sessions
	list, err := store.AllCompact()
	if err != nil {
		t.Fatalf("AllCompact failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 session, got %d", len(list))
	}

	// Get full session
	sess2, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if sess2.ID != sess.ID {
		t.Errorf("ID mismatch: %s vs %s", sess2.ID, sess.ID)
	}

	// Add a message
	err = store.AddMessage(sess.ID, Message{Role: "user", Content: "Hello, world!"})
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}
	sess3, _ := store.Get(sess.ID)
	if len(sess3.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(sess3.Messages))
	}

	// Pin
	err = store.Pin(sess.ID, true)
	if err != nil {
		t.Fatalf("Pin failed: %v", err)
	}
	sess4, _ := store.Get(sess.ID)
	if !sess4.Pinned {
		t.Error("expected pinned=true")
	}

	// Search
	results, err := store.Search("Hello")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search: expected 1 result, got %d", len(results))
	}

	// Title update
	err = store.UpdateTitle(sess.ID)
	if err != nil {
		t.Fatalf("UpdateTitle failed: %v", err)
	}
	sess5, _ := store.Get(sess.ID)
	if sess5.Title != "Hello, world!" {
		t.Errorf("expected title 'Hello, world!', got %q", sess5.Title)
	}

	// Delete
	err = store.Delete(sess.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.Get(sess.ID)
	if err == nil {
		t.Error("expected error getting deleted session")
	}

	t.Log("all session store tests passed")
}

func TestSessionPersistence(t *testing.T) {
	tmp := t.TempDir()

	// Create and save
	store1, err := NewStore(tmp)
	if err != nil {
		t.Fatalf("NewStore1 failed: %v", err)
	}
	sess, _ := store1.New("/workspace", "claude-model")
	store1.AddMessage(sess.ID, Message{Role: "user", Content: "persistent test"})

	// Reopen store
	store2, err := NewStore(tmp)
	if err != nil {
		t.Fatalf("NewStore2 failed: %v", err)
	}
	list, _ := store2.AllCompact()
	if len(list) != 1 {
		t.Errorf("expected 1 session after reopen, got %d", len(list))
	}
	sess2, _ := store2.Get(sess.ID)
	if len(sess2.Messages) != 1 {
		t.Errorf("expected 1 message after reopen, got %d", len(sess2.Messages))
	}
	t.Log("persistence test passed")
}

func TestSessionHash(t *testing.T) {
	tmp := t.TempDir()
	s, _ := NewStore(tmp)
	sess, _ := s.New("/ws", "model")
	s.AddMessage(sess.ID, Message{Role: "user", Content: "test"})
	h1, _ := s.Hash(sess.ID)
	if h1 == "" {
		t.Error("expected non-empty hash")
	}
	// Hash should be stable
	h2, _ := s.Hash(sess.ID)
	if h1 != h2 {
		t.Error("hash should be stable")
	}
	// Different content = different hash
	s.AddMessage(sess.ID, Message{Role: "user", Content: "more"})
	h3, _ := s.Hash(sess.ID)
	if h1 == h3 {
		t.Error("hash should change after message added")
	}
	t.Logf("hash test passed: %s -> %s", h1, h3)
}
