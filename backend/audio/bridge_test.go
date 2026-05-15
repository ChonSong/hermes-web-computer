package audio

import (
	"sync"
	"testing"
)

func TestNewBridge(t *testing.T) {
	b := NewBridge("")
	if b == nil {
		t.Fatal("NewBridge returned nil")
	}
	if b.url != "ws://localhost:11235/api/chat" {
		t.Errorf("expected default url, got %s", b.url)
	}
	if b.sessions == nil {
		t.Error("sessions map should be initialized")
	}
}

func TestNewBridgeCustomURL(t *testing.T) {
	b := NewBridge("ws://example.com/audio")
	if b.url != "ws://example.com/audio" {
		t.Errorf("expected custom url, got %s", b.url)
	}
}

func TestBridge_SessionManagement(t *testing.T) {
	b := NewBridge("")

	// StartSession
	b.StartSession("test-session")
	b.mu.Lock()
	s, ok := b.sessions["test-session"]
	b.mu.Unlock()
	if !ok {
		t.Fatal("session not registered")
	}
	if !s.Active {
		t.Error("session should be active")
	}
	if s.ID != "test-session" {
		t.Errorf("session id = %s, want test-session", s.ID)
	}

	// StopSession
	b.StopSession("test-session")
	b.mu.Lock()
	_, ok = b.sessions["test-session"]
	b.mu.Unlock()
	if ok {
		t.Error("session should be removed after StopSession")
	}
}

func TestBridge_HasConnected(t *testing.T) {
	b := NewBridge("")
	if b.HasConnected() {
		t.Error("new bridge should not report connected")
	}
}

func TestBridge_Interrupt_NoConnection(t *testing.T) {
	b := NewBridge("")
	// Should not error when conn is nil
	err := b.Interrupt("any-session")
	if err != nil {
		t.Errorf("Interrupt with no connection: %v", err)
	}
}

func TestBridge_SendText_NoConnection(t *testing.T) {
	b := NewBridge("")
	err := b.SendText("any-session", "hello")
	if err == nil {
		t.Error("expected error when sending text with no connection")
	}
}

func TestBridge_RelayAudio_NoConnection(t *testing.T) {
	b := NewBridge("")
	_, err := b.RelayAudio("any-session", []byte{1, 2, 3})
	if err == nil {
		t.Error("expected error when relaying audio with no connection")
	}
}

func TestBridge_ConcurrentSessionAccess(t *testing.T) {
	b := NewBridge("")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sid := string(rune('a' + id))
			b.StartSession(sid)
			b.mu.Lock()
			_, ok := b.sessions[sid]
			b.mu.Unlock()
			if !ok {
				t.Errorf("session %s not found", sid)
			}
			b.StopSession(sid)
		}(i)
	}
	wg.Wait()
}