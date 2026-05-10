package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

// Envelope is the JSON-RPC message format for the single WebSocket multiplexer.
type Envelope struct {
	Protocol string          `json:"protocol"` // "ui", "agent", or "audio"
	Method   string          `json:"method"`
	Params   json.RawMessage `json:"params,omitempty"`
	ID       string          `json:"id"`
	Ts       int64           `json:"ts"`
}

// Event is a server-to-client push message.
type Event struct {
	Protocol string          `json:"protocol"`
	Event    string          `json:"event"`
	Data     json.RawMessage `json:"data,omitempty"`
	Ts       int64           `json:"ts"`
}

// Multiplexer routes WebSocket messages by protocol tag.
type Multiplexer struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// Session represents a single WebSocket connection.
type Session struct {
	mu   sync.Mutex
	ws   *websocket.Conn
	send chan Event
	done chan struct{}
}

func NewMultiplexer() *Multiplexer {
	return &Multiplexer{
		sessions: make(map[string]*Session),
	}
}

func (m *Multiplexer) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", m.HandleWebSocket)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	return mux
}

func (m *Multiplexer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "closed")

	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	sess := &Session{
		ws:   conn,
		send: make(chan Event, 256),
		done: make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[sessionID] = sess
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.sessions, sessionID)
		m.mu.Unlock()
	}()

	log.Printf("session %s connected", sessionID)

	// Send initial layout state
	sess.Send(Event{
		Protocol: "ui",
		Event:    "layout.initial",
		Data:     json.RawMessage(`{"layout_version":1,"tree":{"id":"root","type":"leaf","content":"welcome"}}`),
		Ts:       time.Now().UnixMilli(),
	})

	ctx := conn.ReadLimitContext(context.Background())
	go sess.readLoop(ctx, m)
	sess.writeLoop(ctx)

	<-sess.done
	log.Printf("session %s disconnected", sessionID)
}

func (s *Session) readLoop(ctx context.Context, m *Multiplexer) {
	defer close(s.done)
	for {
		_, msg, err := s.ws.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			log.Printf("read error: %v", err)
			return
		}

		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			log.Printf("json unmarshal error: %v", err)
			continue
		}

		env.Ts = time.Now().UnixMilli()
		m.route(env)
	}
}

func (s *Session) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case event := <-s.send:
			event.Ts = time.Now().UnixMilli()
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("marshal error: %v", err)
				continue
			}
			err = s.ws.Write(ctx, websocket.MessageText, data)
			if err != nil {
				log.Printf("write error: %v", err)
				return
			}
		}
	}
}

func (s *Session) Send(event Event) {
	select {
	case s.send <- event:
	default:
		log.Printf("send buffer full, dropping event")
	}
}

func (m *Multiplexer) route(env Envelope) {
	log.Printf("routing: protocol=%s method=%s id=%s", env.Protocol, env.Method, env.ID)
	// TODO: route to appropriate handler based on protocol tag
	// "ui" -> state/layout handler
	// "agent" -> PTY supervisor
	// "audio" -> Fun-Audio-Chat bridge
}
