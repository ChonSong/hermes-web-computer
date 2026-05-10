// Package audio bridges the WebSocket multiplexer to Fun-Audio-Chat's
// native binary protocol on localhost:11235.
package audio

import (
	"context"
	"log"
	"sync"

	"nhooyr.io/websocket"
)

// Bridge relays audio between the client and Fun-Audio-Chat server.
type Bridge struct {
	mu      sync.Mutex
	conn    *websocket.Conn // connection to Fun-Audio-Chat
	sessions map[string]*AudioSession
}

// AudioSession tracks an active audio stream.
type AudioSession struct {
	ID        string
	SessionID string
	Active    bool
}

func NewBridge() *Bridge {
	return &Bridge{
		sessions: make(map[string]*AudioSession),
	}
}

// Connect establishes a WebSocket connection to the Fun-Audio-Chat server.
func (b *Bridge) Connect(ctx context.Context, addr string) error {
	conn, _, err := websocket.Dial(ctx, addr, nil)
	if err != nil {
		return err
	}
	b.conn = conn
	log.Printf("connected to Fun-Audio-Chat at %s", addr)
	return nil
}

// Relay sends an audio chunk to Fun-Audio-Chat and returns the response.
func (b *Bridge) Relay(sessionID string, data []byte) ([]byte, error) {
	// TODO: translate JSON-RPC envelope to Fun-Audio-Chat binary protocol
	// Fun-Audio-Chat protocol: MessageType (1 byte) + payload
	// HANDSHAKE=0x00, AUDIO=0x01, TEXT=0x02, CONTROL=0x03, etc.
	return nil, nil
}

// Interrupt sends a PAUSE control message to abort audio generation mid-inference.
func (b *Bridge) Interrupt(sessionID string) error {
	// TODO: send ControlMessage{PAUSE} to Fun-Audio-Chat
	return nil
}
