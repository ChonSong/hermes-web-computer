// Package audio bridges the WebSocket multiplexer to Fun-Audio-Chat's
// native binary protocol on localhost:11235.
package audio

import (
	"context"
	"fmt"
	"log"
	"sync"

	"nhooyr.io/websocket"
)

// Bridge relays audio between the client and Fun-Audio-Chat server.
type Bridge struct {
	mu       sync.Mutex
	conn     *websocket.Conn // connection to Fun-Audio-Chat
	url      string
	sessions map[string]*AudioSession
}

// AudioSession tracks an active audio stream.
type AudioSession struct {
	ID     string
	Active bool
	Client *websocket.Conn
}

// NewBridge creates a new audio bridge targeting the given URL.
func NewBridge(url string) *Bridge {
	if url == "" {
		url = "ws://localhost:11235/api/chat"
	}
	return &Bridge{url: url, sessions: make(map[string]*AudioSession)}
}

// Connect establishes a WebSocket connection to the Fun-Audio-Chat server.
func (b *Bridge) Connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, b.url, nil)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.conn = conn
	b.mu.Unlock()
	log.Printf("connected to Fun-Audio-Chat at %s", b.url)
	return nil
}

// RelayAudio sends an audio chunk to Fun-Audio-Chat and returns the response.
// Fun-Audio-Chat binary protocol: MessageType (1 byte) + length (2 bytes big-endian) + payload
func (b *Bridge) RelayAudio(sessionID string, opusChunk []byte) ([]byte, error) {
	payload := make([]byte, 3+len(opusChunk))
	payload[0] = 0x01 // AUDIO
	payload[1] = byte(len(opusChunk) >> 8)
	payload[2] = byte(len(opusChunk))
	copy(payload[3:], opusChunk)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.conn == nil {
		return nil, fmt.Errorf("not connected to Fun-Audio-Chat")
	}

	// Send binary frame
	err := b.conn.Write(context.Background(), websocket.MessageBinary, payload)
	if err != nil {
		return nil, err
	}

	// Read response (binary frame)
	_, resp, err := b.conn.Read(context.Background())
	if err != nil {
		return nil, err
	}

	// Response format: MessageType (1 byte) + payload
	if len(resp) < 1 {
		return nil, fmt.Errorf("empty response")
	}

	msgType := resp[0]
	switch msgType {
	case 0x01: // AUDIO
		if len(resp) >= 3 {
			return resp[3:], nil
		}
		return nil, fmt.Errorf("audio response too short")
	case 0x02: // TEXT
		if len(resp) >= 3 {
			return resp[3:], nil
		}
		return nil, fmt.Errorf("text response too short")
	case 0x05: // ERROR
		return nil, fmt.Errorf("Fun-Audio-Chat error: %s", string(resp[3:]))
	default:
		return resp, nil
	}
}

// Interrupt sends a PAUSE control message to abort audio generation mid-inference.
func (b *Bridge) Interrupt(sessionID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn == nil {
		return nil
	}
	// CONTROL + PAUSE = 0x03, 0x02
	return b.conn.Write(context.Background(), websocket.MessageBinary, []byte{0x03, 0x02})
}

// SendText sends a text message to Fun-Audio-Chat.
func (b *Bridge) SendText(sessionID string, text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn == nil {
		return fmt.Errorf("not connected")
	}
	// TEXT message: MessageType (1 byte) + length (2 bytes big-endian) + payload
	payload := make([]byte, 3+len(text))
	payload[0] = 0x02 // TEXT
	payload[1] = byte(len(text) >> 8)
	payload[2] = byte(len(text))
	copy(payload[3:], text)
	return b.conn.Write(context.Background(), websocket.MessageBinary, payload)
}
