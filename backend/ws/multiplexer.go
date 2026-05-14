package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"hermes-web-computer/backend/audio"
	"hermes-web-computer/backend/browser"
	"hermes-web-computer/backend/layout"
	"hermes-web-computer/backend/pty"
	"hermes-web-computer/backend/security"
	"hermes-web-computer/backend/session"
	"hermes-web-computer/backend/state"
	"hermes-web-computer/backend/telemetry"

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
	mu         sync.RWMutex
	sessions   map[string]*Session
	supervisor *pty.Supervisor
	state      *state.SessionState
	layout     *layout.LayoutTree
	enforcer   *security.Enforcer
	telemetry  *telemetry.RingBuffer
	syncer     *telemetry.Syncer
	audio      *audio.Bridge
	browser    *browser.Manager
	hermesURL  string // Hermes Agent API endpoint
	httpClient *http.Client
	contextMgr *ContextManager // tracks focused tile for agent context awareness
	sessionStore *session.Store
}

// Session represents a single WebSocket connection.
type Session struct {
	mu        sync.Mutex
	ws        *websocket.Conn
	send      chan Event
	ptyID     string
	done      chan struct{}
}

func NewMultiplexer() *Multiplexer {
	m := &Multiplexer{
		sessions:   make(map[string]*Session),
		supervisor: pty.NewSupervisor(),
		state: &state.SessionState{
			LayoutVersion: 1,
			Tree: state.LayoutTree{
				ID:      "root",
				Type:    "leaf",
				Content: "welcome",
			},
			AgentState:   "idle",
			ResumePolicy: "B",
		},
		layout:     layout.NewRoot("welcome"),
		enforcer:   security.NewEnforcer(),
		browser:    browser.NewManager(),
		contextMgr: NewContextManager(),
		hermesURL:  os.Getenv("HERMES_API_URL"),
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	if m.hermesURL == "" {
		m.hermesURL = "http://localhost:8642"
	}
	// Load security config (use defaults if file not found)
	homeDir, _ := os.UserHomeDir()
	if err := m.enforcer.LoadConfig(homeDir + "/.agent-os/security.yaml"); err != nil {
		m.enforcer.UseDefaults()
	}
	// Init telemetry
	tb, err := telemetry.NewRingBuffer("/agent/.telemetry/events.jsonl", 100)
	if err == nil {
		m.telemetry = tb
		m.syncer = telemetry.NewSyncer(tb, "")
	}
	// Init session store
	homeDir, _ = os.UserHomeDir()
	storePath := os.Getenv("AGENT_OS_STATE_DIR")
	if storePath == "" {
		storePath = filepath.Join(homeDir, ".agent-os")
	}
	store, err := session.NewStore(storePath)
	if err != nil {
		log.Printf("session store init error: %v (continuing without sessions)", err)
	} else {
		m.sessionStore = store
		log.Printf("session store initialized at %s", storePath)
	}
	return m
}

// SetSessionStore attaches a session store to the multiplexer.
func (m *Multiplexer) SetSessionStore(store *session.Store) {
	m.mu.Lock()
	m.sessionStore = store
	m.mu.Unlock()
}

// SetAudioBridge attaches the audio bridge to the multiplexer.
func (m *Multiplexer) SetAudioBridge(bridge *audio.Bridge) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audio = bridge
	// Connect in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := bridge.Connect(ctx); err != nil {
			log.Printf("audio bridge connect error: %v", err)
		}
	}()
}

// GetTelemetrySyncer returns the telemetry syncer for manual start.
func (m *Multiplexer) GetTelemetrySyncer() *telemetry.Syncer {
	return m.syncer
}

func (m *Multiplexer) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", m.HandleWebSocket)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	// Serve static frontend files - check absolute path first to avoid stale dist dirs
	distPaths := []string{
		"/opt/data/hermes-web-computer/frontend/dist",
		"../frontend/dist",
		"../../frontend/dist",
	}
	for _, distPath := range distPaths {
		if _, err := os.Stat(distPath); err == nil {
			log.Printf("serving static files from %s", distPath)
			fs := http.FileServer(http.Dir(distPath))
			mux.Handle("/", fs)
			break
		}
	}
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
		// Clean up PTY on disconnect
		if sess.ptyID != "" {
			m.supervisor.Signal(sess.ptyID, syscall.SIGTERM)
		}
	}()

	log.Printf("session %s connected", sessionID)

	// Start a PTY session for this connection
	ptyID := fmt.Sprintf("pty_%d", time.Now().UnixNano())
	cmd := exec.Command("bash", "-i")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptySession, err := m.supervisor.Start(ptyID, cmd)
	if err != nil {
		log.Printf("failed to start PTY: %v", err)
	} else {
		sess.ptyID = ptyID
		// Read from PTY and forward to client
		go m.forwardPTYOutput(sess, ptySession)
	}

	// Send initial layout state
	// Reset layout to fresh leaf for each new session so splits work
	m.layout = layout.NewRoot("xterm")
	m.sendEvent(sess, Event{
		Protocol: "ui",
		Event:    "layout.initial",
		Data:     json.RawMessage(fmt.Sprintf(`{"layout_version":1,"tree":{"id":"root","type":"leaf","content":"xterm","pty_id":"%s"}}`, ptyID)),
	})

	// Telemetry: session connected
	if m.telemetry != nil {
		m.telemetry.Write(telemetry.Event{SessionID: sessionID, Type: "session.connected"})
	}

	ctx := context.Background()
	go sess.readLoop(ctx, m, sessionID)
	sess.writeLoop(ctx)

	<-sess.done
	log.Printf("session %s disconnected", sessionID)
}

func (m *Multiplexer) forwardPTYOutput(sess *Session, ptySession *pty.PTYSession) {
	for data := range ptySession.Output {
		// Escape for JSON
		escaped, _ := json.Marshal(string(data))
		m.sendEvent(sess, Event{
			Protocol: "agent",
			Event:    "pty.output",
			Data:     json.RawMessage(fmt.Sprintf(`{"pty_id":"%s","data":%s}`, ptySession.ID, escaped)),
		})
	}
}

func (m *Multiplexer) sendEvent(sess *Session, event Event) {
	sess.Send(event)
}

func (s *Session) readLoop(ctx context.Context, m *Multiplexer, sessionID string) {
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
		m.route(env, s, sessionID)
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

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (m *Multiplexer) route(env Envelope, sess *Session, sessionID string) {
	log.Printf("routing: protocol=%s method=%s id=%s", env.Protocol, env.Method, env.ID)

	switch env.Protocol {
	case "ui":
		m.routeUI(env, sess, sessionID)
	case "agent":
		m.routeAgent(env, sess, sessionID)
	case "audio":
		m.routeAudio(env, sess, sessionID)
	}
}

func (m *Multiplexer) routeUI(env Envelope, sess *Session, sessionID string) {
	switch env.Method {

	case "fs.list":
		m.handleFSList(sess, env.Params)

	case "fs.read":
		m.handleFSRead(sess, env.Params)

	case "fs.write":
		m.handleFSWrite(sess, env.Params)

	case "fs.stat":
		m.handleFSStat(sess, env.Params)

	case "apps.list":
		m.handleAppsList(sess)

	case "apps.launch":
		m.handleAppsLaunch(sess, env.Params)

	case "interrupt":
		// Handle Shift+Space interrupt
		if sess.ptyID != "" {
			// 1. Save checkpoint
			buf, _ := m.supervisor.Checkpoint(sess.ptyID)
			// 2. Signal SIGINT
			m.supervisor.Signal(sess.ptyID, syscall.SIGINT)
			// 3. Update state
			m.state.AgentState = "paused"
			// 4. Send amber border event
			m.sendEvent(sess, Event{
				Protocol: "ui",
				Event:    "border.state",
				Data:     json.RawMessage(`{"color":"amber","state":"paused"}`),
			})
			m.sendEvent(sess, Event{
				Protocol: "ui",
				Event:    "agent.paused",
				Data:     json.RawMessage(fmt.Sprintf(`{"policy":"%s","checkpoint_size":%d}`, m.state.ResumePolicy, len(buf))),
			})
			log.Printf("interrupt handled for session %s", sess.ptyID)

			// Telemetry
			if m.telemetry != nil {
				m.telemetry.Write(telemetry.Event{SessionID: sessionID, Type: "interrupt"})
			}
		}

	case "layout.update":
		var op layout.Op
		if err := json.Unmarshal(env.Params, &op); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		delta, err := m.layout.Apply(op)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.state.LayoutVersion++
		// Send delta with full tree to client (simpler than delta ops for now)
		// The client will use the tree field to replace its layout entirely
		type deltaResponse struct {
			LayoutVersion int64           `json:"layout_version"`
			Tree          *layout.LayoutTree `json:"tree"`
			Ops           []layout.Op      `json:"ops"`
		}
		resp := deltaResponse{
			LayoutVersion: m.state.LayoutVersion,
			Tree:          m.layout,
			Ops:           delta,
		}
		respBytes, _ := json.Marshal(resp)
		m.sendEvent(sess, Event{
			Protocol: "ui",
			Event:    "layout.delta",
			Data:     json.RawMessage(respBytes),
		})
		// Telemetry
		if m.telemetry != nil {
			m.telemetry.Write(telemetry.Event{SessionID: sessionID, Type: "layout.update", Command: op.Op})
		}

	case "approval.grant":
		var params struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			return
		}
		if m.enforcer.ValidateAndConsume(params.Token) {
			// Execute the approved command
			cmd := m.enforcer.GetTokenCommand(params.Token)
			if ptyFile := m.supervisor.PTY(sess.ptyID); ptyFile != nil {
				ptyFile.Write([]byte(cmd))
			}
			m.sendEvent(sess, Event{Protocol: "ui", Event: "approval.granted"})
			if m.telemetry != nil {
				m.telemetry.Write(telemetry.Event{SessionID: sessionID, Type: "approval.granted", Token: params.Token})
			}
		}

	case "dashboard.stats":
		stats := map[string]interface{}{
			"total_sessions":  len(m.sessions),
			"active_sessions": len(m.sessions),
			"uptime_seconds":  time.Since(time.Now().Add(-time.Hour)).Seconds(), // placeholder
			"timestamp":       time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "dashboard.stats.response", Data: json.RawMessage(mustMarshal(stats))})

	case "analytics.get":
		var params struct {
			Days int `json:"days"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if params.Days == 0 {
			params.Days = 7
		}
		result := map[string]interface{}{
			"totals": map[string]interface{}{
				"total_input":          0,
				"total_output":         0,
				"total_sessions":       len(m.sessions),
				"total_api_calls":      0,
				"total_estimated_cost": nil,
			},
			"daily":     []interface{}{},
			"by_model":  []interface{}{},
			"skills":    map[string]interface{}{"top_skills": []interface{}{}},
			"period":    params.Days,
			"timestamp": time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "analytics.result", Data: json.RawMessage(mustMarshal(result))})

	case "system.info":
		info := map[string]interface{}{
			"version":    "v1.0.0",
			"go_version": "1.26",
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
			"timestamp":  time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "system.info.response", Data: json.RawMessage(mustMarshal(info))})

	case "system.resources":
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		res := map[string]interface{}{
			"memory_alloc_mb":   float64(memStats.Alloc) / 1024 / 1024,
			"memory_total_mb":   float64(memStats.TotalAlloc) / 1024 / 1024,
			"goroutines":        runtime.NumGoroutine(),
			"num_cpu":           runtime.NumCPU(),
			"gc_pause_ns":       memStats.PauseTotalNs,
			"timestamp":         time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "system.resources.response", Data: json.RawMessage(mustMarshal(res))})

	case "system.services":
		services := map[string]interface{}{
			"services": []map[string]interface{}{
				{"name": "websocket", "status": "running"},
				{"name": "pty", "status": "running"},
				{"name": "browser", "status": "available"},
				{"name": "audio", "status": func() string { if m.audio != nil { return "available" }; return "unavailable" }()},
			},
			"timestamp": time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "system.services.response", Data: json.RawMessage(mustMarshal(services))})

	case "observability.status":
		status := map[string]interface{}{
			"connected": m.telemetry != nil,
			"timestamp": time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "observability.status", Data: json.RawMessage(mustMarshal(status))})

	case "fs.delete":
		var params struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "fs.delete.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if err := os.RemoveAll(params.Path); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "fs.delete.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		sess.Send(Event{Protocol: "ui", Event: "fs.delete.success", Data: json.RawMessage(fmt.Sprintf(`{"path":%s}`, mustMarshal(params.Path)))})
		if m.telemetry != nil {
			m.telemetry.Write(telemetry.Event{SessionID: sessionID, Type: "fs.delete", Command: params.Path})
		}

	case "ui.focus.change":
		m.handleFocusChange(sess, env.Params, sessionID)

	// ---- Session management ----
	case "session.new":
		var params struct {
			Workspace string `json:"workspace"`
			Model     string `json:"model"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if params.Workspace == "" {
			home, _ := os.UserHomeDir()
			params.Workspace = filepath.Join(home, "workspace")
		}
		if params.Model == "" {
			params.Model = "hermes-agent"
		}
		if m.sessionStore == nil {
			m.sendEvent(sess, Event{Protocol: "ui", Event: "session.new.error", Data: json.RawMessage(`{"message":"session store not available"}`)})
			return
		}
		storeSession, err := m.sessionStore.New(params.Workspace, params.Model)
		if err != nil {
			m.sendEvent(sess, Event{Protocol: "ui", Event: "session.new.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "session.new.ok", Data: json.RawMessage(mustMarshal(storeSession.Compact()))})

	case "session.list":
		if m.sessionStore == nil {
			sess.Send(Event{Protocol: "ui", Event: "error", Data: json.RawMessage(`{"message":"session store not available"}`)})
			return
		}
		list, err := m.sessionStore.AllCompact()
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "session.list", Data: json.RawMessage(mustMarshal(map[string]interface{}{"sessions": list}))})

	case "session.get":
		var params struct {
			ID string `json:"id"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if m.sessionStore == nil || params.ID == "" {
			return
		}
		s, err := m.sessionStore.Get(params.ID)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "session.get.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "session.get", Data: json.RawMessage(mustMarshal(s))})

	case "session.delete":
		var params struct {
			ID string `json:"id"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if m.sessionStore == nil || params.ID == "" {
			return
		}
		if err := m.sessionStore.Delete(params.ID); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "session.delete.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "session.delete.ok", Data: json.RawMessage(fmt.Sprintf(`{"id":%s}`, mustMarshal(params.ID)))})

	case "session.update":
		var params struct {
			ID     string `json:"id"`
			Pinned *bool  `json:"pinned,omitempty"`
			Title  string `json:"title,omitempty"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if m.sessionStore == nil || params.ID == "" {
			return
		}
		if params.Pinned != nil {
			m.sessionStore.Pin(params.ID, *params.Pinned)
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "session.update.ok", Data: json.RawMessage(mustMarshal(params))})
	}
}

func (m *Multiplexer) routeAgent(env Envelope, sess *Session, sessionID string) {
	switch env.Method {
	case "pty.write":
		var params struct {
			Data string `json:"data"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			log.Printf("pty.write unmarshal error: %v", err)
			return
		}

		// Security check: classify the command
		tier, err := m.enforcer.Classify(params.Data, "/agent/workspace")
		if err != nil {
			log.Printf("security classify error: %v", err)
			m.sendEvent(sess, Event{Protocol: "agent", Event: "security.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}

		log.Printf("pty.write: tier=%s data=%q", tier, params.Data)

		switch tier {
		case "safe":
			// Write directly
			if ptyFile := m.supervisor.PTY(sess.ptyID); ptyFile != nil {
				n, err := ptyFile.Write([]byte(params.Data))
				if err != nil {
					log.Printf("pty write error: %v", err)
				} else {
					log.Printf("pty wrote %d bytes", n)
				}
			} else {
				log.Printf("pty file not found for session %s", sess.ptyID)
			}
		case "prompt":
			// Request approval
			token, expiry := m.enforcer.GrantToken(params.Data)
			m.sendEvent(sess, Event{
				Protocol: "ui", Event: "approval.required",
				Data: json.RawMessage(fmt.Sprintf(`{"command":"%s","token":"%s","expires_at":%d}`, params.Data, token, expiry.Unix())),
			})
		case "block":
			m.sendEvent(sess, Event{Protocol: "ui", Event: "command.blocked", Data: json.RawMessage(fmt.Sprintf(`{"command":"%s"}`, params.Data))})
			// Red border
			m.sendEvent(sess, Event{Protocol: "ui", Event: "border.state", Data: json.RawMessage(`{"color":"red","state":"blocked"}`)})
		}

		if m.telemetry != nil {
			trunc := params.Data
			if len(trunc) > 100 {
				trunc = trunc[:100]
			}
			m.telemetry.Write(telemetry.Event{SessionID: sessionID, Type: "pty.write", Command: trunc})
		}

	case "tool.execute":
		var params struct {
			SessionID string                 `json:"session_id"`
			ToolName  string                 `json:"tool_name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			var emptyID string
			data, _ := json.Marshal(map[string]interface{}{"session_id": emptyID, "error": err.Error()})
			m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.error", Data: data})
			return
		}
		go m.handleToolExecute(sess, sessionID, params.SessionID, params.ToolName, params.Arguments)

	case "browser.navigate":
		var params struct {
			SessionID string `json:"session_id"`
			URL       string `json:"url"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			log.Printf("browser.navigate unmarshal error: %v", err)
			return
		}
		inst := m.browser.GetInstance(params.SessionID)
		if inst == nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(`{"message":"browser instance not found"}`)})
			return
		}
		if err := inst.Navigate(params.URL); err != nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		url := inst.GetURL()
		screenshot, err := inst.Screenshot()
		if err != nil {
			log.Printf("browser.screenshot error: %v", err)
		}
		sess.Send(Event{Protocol: "agent", Event: "browser.navigated", Data: json.RawMessage(fmt.Sprintf(`{"session_id":%s,"url":%s,"screenshot":%s}`, mustMarshal(params.SessionID), mustMarshal(url), mustMarshal(screenshot)))})
		log.Printf("browser navigated to %s", params.URL)

	case "browser.screenshot":
		var params struct {
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			log.Printf("browser.screenshot unmarshal error: %v", err)
			return
		}
		inst := m.browser.GetInstance(params.SessionID)
		if inst == nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(`{"message":"browser instance not found"}`)})
			return
		}
		screenshot, err := inst.Screenshot()
		if err != nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		sess.Send(Event{Protocol: "agent", Event: "browser.screenshot.response", Data: json.RawMessage(fmt.Sprintf(`{"session_id":%s,"screenshot":%s}`, mustMarshal(params.SessionID), mustMarshal(screenshot)))})

	case "browser.click":
		var params struct {
			SessionID string  `json:"session_id"`
			X         float64 `json:"x"`
			Y         float64 `json:"y"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			log.Printf("browser.click unmarshal error: %v", err)
			return
		}
		inst := m.browser.GetInstance(params.SessionID)
		if inst == nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(`{"message":"browser instance not found"}`)})
			return
		}
		if err := inst.Click(params.X, params.Y); err != nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		sess.Send(Event{Protocol: "agent", Event: "browser.clicked"})

	case "browser.input":
		var params struct {
			SessionID string `json:"session_id"`
			Text      string `json:"text"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			log.Printf("browser.input unmarshal error: %v", err)
			return
		}
		inst := m.browser.GetInstance(params.SessionID)
		if inst == nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(`{"message":"browser instance not found"}`)})
			return
		}
		if err := inst.Input(params.Text); err != nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		sess.Send(Event{Protocol: "agent", Event: "browser.input.done"})

	case "browser.back":
		var params struct {
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			log.Printf("browser.back unmarshal error: %v", err)
			return
		}
		inst := m.browser.GetInstance(params.SessionID)
		if inst == nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(`{"message":"browser instance not found"}`)})
			return
		}
		if err := inst.GoBack(); err != nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		url := inst.GetURL()
		screenshot, err := inst.Screenshot()
		if err != nil {
			log.Printf("browser.screenshot error: %v", err)
		}
		sess.Send(Event{Protocol: "agent", Event: "browser.navigated", Data: json.RawMessage(fmt.Sprintf(`{"session_id":%s,"url":%s,"screenshot":%s}`, mustMarshal(params.SessionID), mustMarshal(url), mustMarshal(screenshot)))})

	case "browser.forward":
		var params struct {
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			log.Printf("browser.forward unmarshal error: %v", err)
			return
		}
		inst := m.browser.GetInstance(params.SessionID)
		if inst == nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(`{"message":"browser instance not found"}`)})
			return
		}
		if err := inst.GoForward(); err != nil {
			sess.Send(Event{Protocol: "agent", Event: "browser.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		url := inst.GetURL()
		screenshot, err := inst.Screenshot()
		if err != nil {
			log.Printf("browser.screenshot error: %v", err)
		}
		sess.Send(Event{Protocol: "agent", Event: "browser.navigated", Data: json.RawMessage(fmt.Sprintf(`{"session_id":%s,"url":%s,"screenshot":%s}`, mustMarshal(params.SessionID), mustMarshal(url), mustMarshal(screenshot)))})

	case "chat.send":
		var params struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			log.Printf("chat.send unmarshal error: %v", err)
			m.sendEvent(sess, Event{Protocol: "agent", Event: "chat.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		log.Printf("chat.send: %q", params.Message)

		// Forward to Hermes Agent API
		go m.handleChatWithHermes(sess, sessionID, params.Message)
	}
}

func (m *Multiplexer) routeAudio(env Envelope, sess *Session, sessionID string) {
	switch env.Method {
	case "audio.start":
		var params struct {
			SessionID string `json:"session_id"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		sid := params.SessionID
		if sid == "" {
			sid = sessionID
		}
		// Register audio session
		if m.audio != nil {
			m.audio.StartSession(sid)
			m.sendEvent(sess, Event{Protocol: "audio", Event: "audio.started", Data: json.RawMessage(fmt.Sprintf(`{"session_id":%s}`, mustMarshal(sid)))})
			log.Printf("audio session started: %s", sid)
		}

	case "audio.stop":
		if m.audio != nil {
			m.audio.StopSession(sessionID)
			m.sendEvent(sess, Event{Protocol: "audio", Event: "audio.stopped"})
			log.Printf("audio session stopped: %s", sessionID)
		}

	case "audio.stream":
		var params struct {
			OpusChunk []byte `json:"opus_chunk"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			return
		}
		if m.audio != nil {
			resp, err := m.audio.RelayAudio(sessionID, params.OpusChunk)
			if err != nil {
				m.sendEvent(sess, Event{Protocol: "audio", Event: "error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
				return
			}
			// Relay response back to client
			m.sendEvent(sess, Event{Protocol: "audio", Event: "response", Data: json.RawMessage(fmt.Sprintf(`{"data":%s}`, mustMarshal(string(resp))))})
		}

	case "audio.interrupt":
		if m.audio != nil {
			if err := m.audio.Interrupt(sessionID); err != nil {
				log.Printf("audio interrupt error: %v", err)
			}
		}

	case "audio.text":
		var params struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			return
		}
		if m.audio != nil {
			if err := m.audio.SendText(sessionID, params.Text); err != nil {
				log.Printf("audio send text error: %v", err)
			}
		}
	}
}

// handleChatWithHermes forwards a chat message to the Hermes Agent API
// and streams the response back to the client via WebSocket.
func (m *Multiplexer) handleChatWithHermes(sess *Session, sessionID string, message string) {
	// Telemetry
	if m.telemetry != nil {
		trunc := message
		if len(trunc) > 100 {
			trunc = trunc[:100]
		}
		m.telemetry.Write(telemetry.Event{SessionID: sessionID, Type: "chat.send", Command: trunc})
	}

	// Build the request to Hermes Agent, including focus context for auto-scoping
	reqBody := map[string]interface{}{
		"message": message,
	}
	// Attach focus context so agent can auto-scope responses
	if scope := m.contextMgr.BuildAgentScope(); scope != "" {
		reqBody["context"] = scope
	}
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "chat.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"Failed to marshal request: %s"}`, err.Error()))})
		return
	}

	// Try the Hermes Agent API endpoint
	req, err := http.NewRequest("POST", m.hermesURL+"/api/chat", bytes.NewReader(reqData))
	if err != nil {
		log.Printf("hermes chat request error: %v", err)
		m.sendEvent(sess, Event{Protocol: "agent", Event: "chat.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"Failed to connect to Hermes agent: %s"}`, err.Error()))})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		log.Printf("hermes api error: %v", err)
		// Fallback: send a friendly message
		m.sendEvent(sess, Event{Protocol: "agent", Event: "chat.reply", Data: json.RawMessage(fmt.Sprintf(`{"message":"Agent is not available (Hermes not running on %s). Your message was: %q","complete":true}`, m.hermesURL, message))})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("hermes api non-200 response: %d - %s", resp.StatusCode, string(body))
		m.sendEvent(sess, Event{Protocol: "agent", Event: "chat.reply", Data: json.RawMessage(fmt.Sprintf(`{"message":"Agent returned status %d: %s","complete":true}`, resp.StatusCode, string(body)))})
		return
	}

	// Read and forward the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("hermes api read error: %v", err)
		m.sendEvent(sess, Event{Protocol: "agent", Event: "chat.error", Data: json.RawMessage(`{"message":"Failed to read agent response"}`)})
		return
	}

	// Try to parse as JSON response from Hermes
	var hermesResp struct {
		Message   string `json:"message"`
		Response  string `json:"response"`
		Text      string `json:"text"`
		Reply     string `json:"reply"`
		Content   string `json:"content"`
		Streaming bool   `json:"streaming"`
	}
	if err := json.Unmarshal(body, &hermesResp); err == nil {
		reply := hermesResp.Message
		if reply == "" {
			reply = hermesResp.Response
		}
		if reply == "" {
			reply = hermesResp.Text
		}
		if reply == "" {
			reply = hermesResp.Reply
		}
		if reply == "" {
			reply = hermesResp.Content
		}
		if reply != "" {
			m.sendEvent(sess, Event{Protocol: "agent", Event: "chat.reply", Data: json.RawMessage(fmt.Sprintf(`{"message":%s,"complete":true}`, mustMarshal(reply)))})
			return
		}
	}

	// If not JSON, treat raw body as text response
	if len(body) > 0 {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "chat.reply", Data: json.RawMessage(fmt.Sprintf(`{"message":%s,"complete":true}`, mustMarshal(string(body))))})
		return
	}

	// Empty response
	m.sendEvent(sess, Event{Protocol: "agent", Event: "chat.reply", Data: json.RawMessage(`{"message":"(empty response)","complete":true}`)})
}

// handleToolExecute calls the Hermes Agent tool.execute API and sends the result back via WebSocket.
func (m *Multiplexer) handleToolExecute(sess *Session, sessionID string, tileSessionID string, toolName string, args map[string]interface{}) {
	argsJSON, _ := json.Marshal(args)

	toolFuncDef := map[string]interface{}{
		"name":        toolName,
		"description": "Execute the " + toolName + " tool",
		"parameters":  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	}
	toolSpec := map[string]interface{}{
		"type":     "function",
		"function": toolFuncDef,
	}
	toolChoice := map[string]interface{}{
		"type":     "function",
		"function": map[string]interface{}{"name": toolName},
	}

	reqBody := map[string]interface{}{
		"model": "hermes-agent",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": fmt.Sprintf("Please execute the tool %q with arguments: %s", toolName, string(argsJSON)),
			},
		},
		"tools":       []map[string]interface{}{toolSpec},
		"tool_choice": toolChoice,
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		data, _ := json.Marshal(map[string]interface{}{"session_id": tileSessionID, "tool_name": toolName, "error": "Failed to marshal request: " + err.Error()})
		m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.error", Data: data})
		return
	}

	hermesURL := m.hermesURL
	if hermesURL == "" {
		hermesURL = "http://localhost:8642"
	}

	req, err := http.NewRequest("POST", hermesURL+"/v1/chat/completions", bytes.NewReader(reqData))
	if err != nil {
		data, _ := json.Marshal(map[string]interface{}{"session_id": tileSessionID, "tool_name": toolName, "error": "Failed to create request: " + err.Error()})
		m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.error", Data: data})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		data, _ := json.Marshal(map[string]interface{}{"session_id": tileSessionID, "tool_name": toolName, "error": "Hermes agent unavailable: " + err.Error()})
		m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.error", Data: data})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		data, _ := json.Marshal(map[string]interface{}{"session_id": tileSessionID, "tool_name": toolName, "error": "Failed to read response: " + err.Error()})
		m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.error", Data: data})
		return
	}

	if resp.StatusCode != http.StatusOK {
		data, _ := json.Marshal(map[string]interface{}{"session_id": tileSessionID, "tool_name": toolName, "error": fmt.Sprintf("Hermes returned status %d: %s", resp.StatusCode, string(body))})
		m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.error", Data: data})
		return
	}

	var completionsResp struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID   string `json:"id"`
					Type string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &completionsResp); err != nil {
		data, _ := json.Marshal(map[string]interface{}{"session_id": tileSessionID, "tool_name": toolName, "error": "Failed to parse response: " + err.Error()})
		m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.error", Data: data})
		return
	}

	if len(completionsResp.Choices) == 0 {
		data, _ := json.Marshal(map[string]interface{}{"session_id": tileSessionID, "tool_name": toolName, "result": nil})
		m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.result", Data: data})
		return
	}

	msg := completionsResp.Choices[0].Message
	var result string

	if len(msg.ToolCalls) > 0 {
		result = msg.ToolCalls[0].Function.Arguments
	} else if msg.Content != "" {
		result = msg.Content
	}

	resultData := map[string]interface{}{"session_id": tileSessionID, "tool_name": toolName, "result": json.RawMessage(mustMarshal(result))}
	data, _ := json.Marshal(resultData)
	m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.result", Data: data})
}

