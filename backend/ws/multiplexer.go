package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"nhooyr.io/websocket"
	"hermes-web-computer/backend/agent"
	"hermes-web-computer/backend/audio"
	"hermes-web-computer/backend/browser"
	"hermes-web-computer/backend/config"
	"hermes-web-computer/backend/docker"
	"hermes-web-computer/backend/layout"
	"hermes-web-computer/backend/mcp"
	"hermes-web-computer/backend/pty"
	"hermes-web-computer/backend/security"
	"hermes-web-computer/backend/session"
	"hermes-web-computer/backend/state"
	"hermes-web-computer/backend/telemetry"
	"hermes-web-computer/backend/xpra"
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
	configMgr   *config.Manager
	dockerMgr   *docker.Manager
	mcpMgr      *mcp.Manager
xpraMgr     *xpra.Manager
	startTime   time.Time // server start time for uptime calculation
}

// SetConfigManager attaches the config manager to the multiplexer.
func (m *Multiplexer) SetConfigManager(cm *config.Manager) {
	m.mu.Lock()
	m.configMgr = cm
	m.mu.Unlock()
}

// SetDockerManager attaches the docker manager to the multiplexer.
func (m *Multiplexer) SetDockerManager(dm *docker.Manager) {
	m.mu.Lock()
	m.dockerMgr = dm
	m.mu.Unlock()
}

// SetXpraManager attaches the Xpra manager to the multiplexer.
func (m *Multiplexer) SetXpraManager(xm *xpra.Manager) {
	m.mu.Lock()
	m.xpraMgr = xm
	m.mu.Unlock()
}

// InitializeXpra initializes the Xpra manager with a display number.
func (m *Multiplexer) InitializeXpra(displayNum int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.xpraMgr != nil {
		return nil
	}
	m.xpraMgr = xpra.New("default", displayNum)
	return nil
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
		startTime:  time.Now(),
	}
	if m.hermesURL == "" {
		m.hermesURL = "http://localhost:8642"
	}
	// Load security config (use defaults if file not found)
	homeDir, _ := os.UserHomeDir()
	securityConfigPath := os.Getenv("HWC_SECURITY_CONFIG")
	if securityConfigPath == "" {
		securityConfigPath = filepath.Join(homeDir, ".hermes", "hermes-web-computer", "security.yaml")
	}
	if err := m.enforcer.LoadConfig(securityConfigPath); err != nil {
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
	storePath := os.Getenv("HWC_STATE_DIR")
	if storePath == "" {
		storePath = filepath.Join(homeDir, ".hermes", "hermes-web-computer")
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
	mux.HandleFunc("/api/system/metrics", ServeMetricsHTTP)
	// Xpra HTML5 proxy — only register if Xpra manager is initialized
	mux.HandleFunc("/api/xpra/", m.handleXpraProxy)
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

// handleXpraProxy proxies /api/xpra/* requests to the Xpra HTML5 server.
func (m *Multiplexer) handleXpraProxy(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	xm := m.xpraMgr
	m.mu.RUnlock()

	if xm == nil || !xm.IsRunning() {
		http.Error(w, "Xpra not available (not installed or not started)", 503)
		return
	}

	proxy := xpra.NewProxyHandler(xm)
	proxy.ServeHTTP(w, r)
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

// tokenEstimate approximates token count from text (rough: 1 token ≈ 4 chars).
func tokenEstimate(text string) int {
	return (len(text) / 4) + 1
}

// providerFromModel extracts the provider name from a model string.
func providerFromModel(model string) string {
	model = strings.ToLower(model)
	switch {
	case strings.Contains(model, "claude"):
		return "anthropic"
	case strings.Contains(model, "gpt-4") || strings.Contains(model, "gpt-3.5") || strings.Contains(model, "o1") || strings.Contains(model, "o1-mini"):
		return "openai"
	case strings.Contains(model, "gemini"):
		return "google"
	case strings.Contains(model, "deepseek"):
		return "deepseek"
	case strings.Contains(model, "groq") || strings.Contains(model, "llama"):
		return "groq"
	case strings.Contains(model, "ollama"):
		return "ollama"
	default:
		return "unknown"
	}
}

// modelFamily extracts a short family name from a model string.
func modelFamily(model string) string {
	model = strings.ToLower(model)
	if strings.Contains(model, "claude-3-5-opus") || strings.Contains(model, "claude-opus") {
		return "claude-opus"
	}
	if strings.Contains(model, "claude-3-5-sonnet") || strings.Contains(model, "claude-sonnet") {
		return "claude-sonnet"
	}
	if strings.Contains(model, "claude-3-5-haiku") || strings.Contains(model, "claude-haiku") {
		return "claude-haiku"
	}
	if strings.Contains(model, "claude-3-opus") {
		return "claude-3-opus"
	}
	if strings.Contains(model, "claude-3-sonnet") {
		return "claude-3-sonnet"
	}
	if strings.Contains(model, "claude-3-haiku") {
		return "claude-3-haiku"
	}
	if strings.Contains(model, "gpt-4o") {
		return "gpt-4o"
	}
	if strings.Contains(model, "gpt-4-turbo") {
		return "gpt-4-turbo"
	}
	if strings.Contains(model, "gpt-4") {
		return "gpt-4"
	}
	if strings.Contains(model, "gpt-3.5-turbo") {
		return "gpt-3.5-turbo"
	}
	if strings.Contains(model, "o1") {
		return "o1"
	}
	if strings.Contains(model, "gemini-2-flash") {
		return "gemini-2-flash"
	}
	if strings.Contains(model, "gemini-1-5-flash") {
		return "gemini-1.5-flash"
	}
	if strings.Contains(model, "gemini-pro") {
		return "gemini-pro"
	}
	if strings.Contains(model, "deepseek") {
		return "deepseek"
	}
	if strings.Contains(model, "llama") {
		return "llama"
	}
	return model
}

// modelPricing returns estimated cost per 1M tokens for a given model.
// Prices are approximate for common models (Anthropic, OpenAI, Google, DeepSeek).
func modelPricing(model string) (input float64, output float64) {
	model = strings.ToLower(model)
	switch {
	case strings.Contains(model, "claude-opus") || strings.Contains(model, "claude-3-5-opus"):
		return 15.0, 75.0
	case strings.Contains(model, "claude-sonnet") || strings.Contains(model, "claude-3-5-sonnet"):
		return 3.0, 15.0
	case strings.Contains(model, "claude-haiku") || strings.Contains(model, "claude-3-5-haiku"):
		return 0.8, 4.0
	case strings.Contains(model, "claude-3-opus"):
		return 15.0, 75.0
	case strings.Contains(model, "claude-3-sonnet"):
		return 3.0, 15.0
	case strings.Contains(model, "claude-3-haiku"):
		return 0.25, 1.25
	case strings.Contains(model, "gpt-4o"):
		return 5.0, 20.0
	case strings.Contains(model, "gpt-4-turbo"):
		return 10.0, 30.0
	case strings.Contains(model, "gpt-4"):
		return 30.0, 60.0
	case strings.Contains(model, "gpt-35-turbo") || strings.Contains(model, "gpt-3.5-turbo"):
		return 0.5, 1.5
	case strings.Contains(model, "o1"):
		return 15.0, 60.0
	case strings.Contains(model, "o1-mini"):
		return 3.0, 12.0
	case strings.Contains(model, "gemini-2-flash"):
		return 0.1, 0.4
	case strings.Contains(model, "gemini-1-5-flash"):
		return 0.075, 0.3
	case strings.Contains(model, "gemini-pro"):
		return 1.25, 5.0
	case strings.Contains(model, "deepseek-chat") || strings.Contains(model, "deepseek-v3"):
		return 0.14, 0.28
	case strings.Contains(model, "deepseek-coder"):
		return 0.14, 0.28
	case strings.Contains(model, "llama-3-1-70b") || strings.Contains(model, "llama-3-1-8b"):
		return 0.0, 0.0 // Groq free tier
	default:
		return 1.0, 2.0
	}
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

	case "fs.rename":
		m.handleFSRename(sess, env.Params)

	case "fs.delete":
		m.handleFSDelete(sess, env.Params)

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
		// Compute real dashboard stats from session store and system info
		hostname, _ := os.Hostname()
		stats := map[string]interface{}{
			"total_sessions":  0,
			"active_sessions": len(m.sessions),
			"uptime_seconds":  int64(time.Since(m.startTime).Seconds()),
			"hostname":        hostname,
			"version":         "v1.3.0",
			"started_at":      m.startTime.Unix(),
			"timestamp":       time.Now().UnixMilli(),
		}
		if m.sessionStore != nil {
			allSessions, _ := m.sessionStore.All()
			stats["total_sessions"] = len(allSessions)
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
		// Compute real analytics from session store + cost ledger (Step 6.2)
		totals := map[string]interface{}{
			"total_input":          0,
			"total_output":         0,
			"total_sessions":       0,
			"total_api_calls":      0,
			"total_estimated_cost": 0.0,
		}
		var daily []interface{}
		var byModel []interface{}
		topSkills := []interface{}{}

		// Track per-model stats for cost ledger (Step 6.2)
		modelStats := make(map[string]map[string]interface{})

		if m.sessionStore != nil {
			allSessions, err := m.sessionStore.All()
			if err == nil {
				totals["total_sessions"] = len(allSessions)
				var totalInput, totalOutput int
				// Group sessions by day for daily breakdown
				dailyMap := make(map[string]map[string]interface{})

				for _, sess := range allSessions {
					// Daily aggregation
					day := time.Unix(sess.CreatedAt, 0).Format("2006-01-02")
					if dailyMap[day] == nil {
						dailyMap[day] = map[string]interface{}{"day": day, "sessions": 0, "input_tokens": 0, "output_tokens": 0}
					}

					for _, msg := range sess.Messages {
						if msg.Role == "user" {
							tokens := tokenEstimate(msg.Content)
							totalInput += tokens
							inputInt := dailyMap[day]["input_tokens"].(int) + tokens
							dailyMap[day]["input_tokens"] = inputInt
						} else if msg.Role == "assistant" {
							tokens := tokenEstimate(msg.Content)
							totalOutput += tokens
							outputInt := dailyMap[day]["output_tokens"].(int) + tokens
							dailyMap[day]["output_tokens"] = outputInt
						}
					}

					// Per-model stats
					model := sess.Model
					if model == "" {
						model = "unknown"
					}
					if modelStats[model] == nil {
						modelStats[model] = map[string]interface{}{
							"model": model, "provider": providerFromModel(model),
							"sessions": 0, "input_tokens": 0, "output_tokens": 0,
							"cache_read_tokens": 0, "api_calls": 0, "estimated_cost": 0.0,
							"last_used_at": int64(0),
						}
					}
					ms := modelStats[model]
					ms["sessions"] = ms["sessions"].(int) + 1
					if sess.UpdatedAt > ms["last_used_at"].(int64) {
						ms["last_used_at"] = sess.UpdatedAt
					}

					// Count API calls from message structure
					var msgInput, msgOutput int
					for _, msg := range sess.Messages {
						if msg.Role == "user" {
							msgInput += tokenEstimate(msg.Content)
						} else if msg.Role == "assistant" {
							msgOutput += tokenEstimate(msg.Content)
						}
					}
					ms["input_tokens"] = ms["input_tokens"].(int) + msgInput
					ms["output_tokens"] = ms["output_tokens"].(int) + msgOutput

					dailyMap[day]["sessions"] = dailyMap[day]["sessions"].(int) + 1
				}

				// Convert dailyMap to sorted slice (newest first)
				type dayEntry struct{ day string; data map[string]interface{} }
				var dailyList []dayEntry
				for day, data := range dailyMap {
					dailyList = append(dailyList, dayEntry{day, data})
				}
				sort.Slice(dailyList, func(i, j int) bool { return dailyList[i].day > dailyList[j].day })
				for _, de := range dailyList {
					daily = append(daily, de.data)
				}

				totals["total_input"] = totalInput
				totals["total_output"] = totalOutput

				// Compute estimated cost per model (Step 6.2)
				var totalCost float64
				for model, ms := range modelStats {
					inp := ms["input_tokens"].(int)
					out := ms["output_tokens"].(int)
					inPrice, outPrice := modelPricing(model)
					cost := (float64(inp)/1_000_000)*inPrice + (float64(out)/1_000_000)*outPrice
					ms["estimated_cost"] = math.Round(cost*100) / 100
					totalCost += cost

					// Detect capabilities from model name
					capabilities := map[string]interface{}{
						"supports_tools":      strings.Contains(model, "gpt-4") || strings.Contains(model, "claude") || strings.Contains(model, "gemini"),
						"supports_vision":    strings.Contains(model, "4o") || strings.Contains(model, "claude-3"),
						"supports_reasoning":  strings.Contains(model, "o1") || strings.Contains(model, "claude-3-5") || strings.Contains(model, "opus"),
						"context_window":       128000,
						"max_output_tokens":   8192,
						"model_family":        modelFamily(model),
					}
					ms["capabilities"] = capabilities
					byModel = append(byModel, ms)
				}
				totals["total_estimated_cost"] = math.Round(totalCost*100) / 100
			}
		}

		result := map[string]interface{}{
			"totals":   totals,
			"daily":    daily,
			"by_model": byModel,
			"skills":   map[string]interface{}{"top_skills": topSkills},
			"period":   params.Days,
			"timestamp": time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "analytics.result", Data: json.RawMessage(mustMarshal(result))})

	case "system.info":
		hostname, _ := os.Hostname()
		info := map[string]interface{}{
			"version":      "v1.3.0",
			"go_version":   runtime.Version(),
			"os":           runtime.GOOS,
			"arch":         runtime.GOARCH,
			"hostname":     hostname,
			"uptime":       int64(time.Since(m.startTime).Seconds()),
			"num_cpu":      runtime.NumCPU(),
			"total_mem_gb": getTotalMemGB(),
			"timestamp":    time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "system.info.result", Data: json.RawMessage(mustMarshal(info))})

	case "system.metrics":
		metrics := globalCollector.FetchMetrics()
		resp, _ := json.Marshal(metrics)
		m.sendEvent(sess, Event{Protocol: "ui", Event: "system.metrics", Data: json.RawMessage(resp)})

	case "system.resources":
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		// Read host memory from /proc/meminfo for real system memory usage
		hostMemUsed, hostMemTotal := readHostMemInfo()
		res := map[string]interface{}{
			"cpu_percent":   getCPUUsage(),
			"mem_used_gb":  hostMemUsed,
			"mem_total_gb": hostMemTotal,
			"mem_percent":  func() float64 { if hostMemTotal > 0 { return (hostMemUsed / hostMemTotal) * 100 }; return 0 }(),
			"disk_used_gb": getDiskUsage(),
			"disk_total_gb": getDiskTotal(),
			"disk_percent": getDiskPercent(),
			"goroutines":   runtime.NumGoroutine(),
			"gc_pause_ns":  memStats.PauseTotalNs,
			"timestamp":    time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "system.resources.result", Data: json.RawMessage(mustMarshal(res))})

	case "system.services":
		// Build real service list with uptime
		var svcList []map[string]interface{}
		svcList = append(svcList, map[string]interface{}{
			"name":    "websocket",
			"running": true,
			"pid":     os.Getpid(),
			"uptime":  int64(time.Since(m.startTime).Seconds()),
		})
		svcList = append(svcList, map[string]interface{}{
			"name":    "pty",
			"running": m.supervisor != nil,
			"uptime":  int64(time.Since(m.startTime).Seconds()),
		})
		svcList = append(svcList, map[string]interface{}{
			"name":    "browser",
			"running": m.browser != nil,
		})
		if m.audio != nil {
			svcList = append(svcList, map[string]interface{}{
				"name":    "audio",
				"running": true,
			})
		} else {
			svcList = append(svcList, map[string]interface{}{
				"name":    "audio",
				"running": false,
			})
		}
		services := map[string]interface{}{
			"services": svcList,
			"timestamp": time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "system.services.result", Data: json.RawMessage(mustMarshal(services))})

	case "observability.status":
		status := map[string]interface{}{
			"connected": m.telemetry != nil,
			"timestamp": time.Now().UnixMilli(),
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "observability.status", Data: json.RawMessage(mustMarshal(status))})

	case "observability.events":
		var params struct {
			Limit int `json:"limit"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if params.Limit == 0 {
			params.Limit = 100
		}
		if params.Limit > 500 {
			params.Limit = 500
		}
		var events []interface{}
		if m.telemetry != nil {
			evts, err := m.telemetry.ReadLast(params.Limit)
			if err == nil {
				for _, e := range evts {
					events = append(events, map[string]interface{}{
						"type":      e.Type,
						"timestamp": time.UnixMilli(e.Ts).Format(time.RFC3339),
						"data": map[string]interface{}{
							"session":  e.SessionID,
							"user":     e.User,
							"policy":   e.Policy,
							"command":  e.Command,
							"tool":     e.Tool,
							"path":      e.Path,
							"outcome":  e.Outcome,
							"drift_score": e.DriftScore,
						},
					})
				}
			}
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "observability.events.result", Data: json.RawMessage(mustMarshal(map[string]interface{}{"events": events, "count": len(events)}))})

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

	// ---- Docker management ----
	case "docker.list":
		if m.dockerMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(`{"message":"docker manager not available"}`)})
			return
		}
		ctx := context.Background()
		containers, err := m.dockerMgr.ListContainers(ctx)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.list.response", Data: json.RawMessage(mustMarshal(map[string]interface{}{"containers": containers}))})

	case "docker.stats":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.ID == "" {
			return
		}
		ctx := context.Background()
		stats, err := m.dockerMgr.GetStats(ctx, params.ID)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.stats.response", Data: json.RawMessage(mustMarshal(stats))})

	case "docker.start":
		var params struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.ID == "" {
			return
		}
		ctx := context.Background()
		if err := m.dockerMgr.StartContainer(ctx, params.ID); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.start.ok", Data: json.RawMessage(fmt.Sprintf(`{"id":%s}`, mustMarshal(params.ID)))})

	case "docker.stop":
		var params struct {
			ID     string `json:"id"`
			Timeout *int   `json:"timeout,omitempty"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.ID == "" {
			return
		}
		ctx := context.Background()
		if err := m.dockerMgr.StopContainer(ctx, params.ID, params.Timeout); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.stop.ok", Data: json.RawMessage(fmt.Sprintf(`{"id":%s}`, mustMarshal(params.ID)))})

	case "docker.restart":
		var params struct {
			ID     string `json:"id"`
			Timeout *int   `json:"timeout,omitempty"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.ID == "" {
			return
		}
		ctx := context.Background()
		if err := m.dockerMgr.RestartContainer(ctx, params.ID, params.Timeout); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.restart.ok", Data: json.RawMessage(fmt.Sprintf(`{"id":%s}`, mustMarshal(params.ID)))})

	case "docker.remove":
		var params struct {
			ID            string `json:"id"`
			Force         bool   `json:"force,omitempty"`
			RemoveVolumes bool   `json:"remove_volumes,omitempty"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.ID == "" {
			return
		}
		ctx := context.Background()
		if err := m.dockerMgr.RemoveContainer(ctx, params.ID, params.Force, params.RemoveVolumes); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.remove.ok", Data: json.RawMessage(fmt.Sprintf(`{"id":%s}`, mustMarshal(params.ID)))})

	case "docker.logs":
		var params struct {
			ID   string `json:"id"`
			Tail string `json:"tail,omitempty"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.ID == "" {
			return
		}
		ctx := context.Background()
		logs, err := m.dockerMgr.GetLogs(ctx, params.ID, params.Tail)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.logs.response", Data: json.RawMessage(fmt.Sprintf(`{"id":%s,"logs":%s}`, mustMarshal(params.ID), mustMarshal(logs)))})

	case "docker.create":
		var params struct {
			Image   string   `json:"image"`
			Name    string   `json:"name,omitempty"`
			Ports   []string `json:"ports,omitempty"`
			EnvVars []string `json:"env_vars,omitempty"`
			Volumes []string `json:"volumes,omitempty"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.Image == "" {
			return
		}
		ctx := context.Background()
		id, err := m.dockerMgr.CreateContainer(ctx, params.Image, params.Name, params.Ports, params.EnvVars, params.Volumes)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.create.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.create.ok", Data: json.RawMessage(fmt.Sprintf(`{"id":%s}`, mustMarshal(id)))})

	case "docker.images":
		if m.dockerMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(`{"message":"docker manager not available"}`)})
			return
		}
		ctx := context.Background()
		images, err := m.dockerMgr.ListImages(ctx)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.images.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.images.ok", Data: json.RawMessage(mustMarshal(map[string]interface{}{"images": images}))})

	case "docker.image.remove":
		var params struct {
			ID    string `json:"id"`
			Force bool   `json:"force,omitempty"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.ID == "" {
			return
		}
		ctx := context.Background()
		if err := m.dockerMgr.RemoveImage(ctx, params.ID, params.Force); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.image.remove.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.image.remove.ok", Data: json.RawMessage(fmt.Sprintf(`{"id":%s}`, mustMarshal(params.ID)))})

	case "docker.image.pull":
		var params struct {
			Image string `json:"image"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.Image == "" {
			return
		}
		ctx := context.Background()
		if err := m.dockerMgr.PullImage(ctx, params.Image); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.image.pull.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.image.pull.ok", Data: json.RawMessage(fmt.Sprintf(`{"image":%s}`, mustMarshal(params.Image)))})

	case "docker.compose.ls":
		if m.dockerMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(`{"message":"docker manager not available"}`)})
			return
		}
		ctx := context.Background()
		projects, err := m.dockerMgr.ListComposeProjects(ctx)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.compose.ls.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.compose.ls.ok", Data: json.RawMessage(mustMarshal(map[string]interface{}{"projects": projects}))})

	case "docker.compose.up":
		var params struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.Path == "" {
			return
		}
		ctx := context.Background()
		if err := m.dockerMgr.ComposeUp(ctx, params.Path, true); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.compose.up.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.compose.up.ok", Data: json.RawMessage(fmt.Sprintf(`{"path":%s}`, mustMarshal(params.Path)))})

	case "docker.compose.down":
		var params struct {
			Path          string `json:"path"`
			RemoveVolumes bool   `json:"remove_volumes,omitempty"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.Path == "" {
			return
		}
		ctx := context.Background()
		if err := m.dockerMgr.ComposeDown(ctx, params.Path, params.RemoveVolumes); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.compose.down.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.compose.down.ok", Data: json.RawMessage(fmt.Sprintf(`{"path":%s}`, mustMarshal(params.Path)))})

	case "docker.compose.stop":
		var params struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(env.Params, &params); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if m.dockerMgr == nil || params.Path == "" {
			return
		}
		ctx := context.Background()
		if err := m.dockerMgr.ComposeStop(ctx, params.Path); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "docker.compose.stop.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "docker.compose.stop.ok", Data: json.RawMessage(fmt.Sprintf(`{"path":%s}`, mustMarshal(params.Path)))})

	// ---- Config management ----
	case "config.get":
		if m.configMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "config.get.error", Data: json.RawMessage(`{"message":"config manager not available"}`)})
			return
		}
		data, err := m.configMgr.ToJSON()
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "config.get.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "config.get", Data: json.RawMessage(data)})

	case "config.set":
		if m.configMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "config.set.error", Data: json.RawMessage(`{"message":"config manager not available"}`)})
			return
		}
		var params struct {
			Key   string      `json:"key"`
			Value interface{} `json:"value"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if params.Key == "" {
			sess.Send(Event{Protocol: "ui", Event: "config.set.error", Data: json.RawMessage(`{"message":"key is required"}`)})
			return
		}
		if err := m.configMgr.Set(params.Key, params.Value); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "config.set.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if err := m.configMgr.Save(); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "config.set.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "config.set.ok", Data: json.RawMessage(mustMarshal(params))})

	case "config.delete":
		if m.configMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "config.delete.error", Data: json.RawMessage(`{"message":"config manager not available"}`)})
			return
		}
		var params struct {
			Key string `json:"key"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if params.Key == "" {
			sess.Send(Event{Protocol: "ui", Event: "config.delete.error", Data: json.RawMessage(`{"message":"key is required"}`)})
			return
		}
		if err := m.configMgr.Delete(params.Key); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "config.delete.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if err := m.configMgr.Save(); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "config.delete.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "config.delete.ok", Data: json.RawMessage(mustMarshal(params))})

	// ---- Env management ----
	case "env.list":
		if m.configMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "env.list.error", Data: json.RawMessage(`{"message":"config manager not available"}`)})
			return
		}
		envs := m.configMgr.EnvList()
		m.sendEvent(sess, Event{Protocol: "ui", Event: "env.list", Data: json.RawMessage(mustMarshal(map[string]interface{}{"env": envs}))})

	case "env.set":
		if m.configMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "env.set.error", Data: json.RawMessage(`{"message":"config manager not available"}`)})
			return
		}
		var params struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if params.Key == "" {
			sess.Send(Event{Protocol: "ui", Event: "env.set.error", Data: json.RawMessage(`{"message":"key is required"}`)})
			return
		}
		if err := m.configMgr.EnvSet(params.Key, params.Value); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "env.set.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if err := m.configMgr.Save(); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "env.set.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "env.set.ok", Data: json.RawMessage(mustMarshal(params))})

	case "env.delete":
		if m.configMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "env.delete.error", Data: json.RawMessage(`{"message":"config manager not available"}`)})
			return
		}
		var params struct {
			Key string `json:"key"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if params.Key == "" {
			sess.Send(Event{Protocol: "ui", Event: "env.delete.error", Data: json.RawMessage(`{"message":"key is required"}`)})
			return
		}
		if err := m.configMgr.EnvDelete(params.Key); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "env.delete.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		if err := m.configMgr.Save(); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "env.delete.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "env.delete.ok", Data: json.RawMessage(mustMarshal(params))})

	// ---- System restart ----
	case "system.restart":
		if m.configMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "system.restart.error", Data: json.RawMessage(`{"message":"config manager not available"}`)})
			return
		}
		if err := m.configMgr.RestartSignal(); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "system.restart.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "system.restart.ok", Data: json.RawMessage(`{"message":"restart signal sent"}`)})

	// ---- Xpra management ----
	case "xpra.start":
		var params struct {
			Display string `json:"display"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if m.xpraMgr == nil {
			// Auto-initialize with display 10
			m.mu.Lock()
			if m.xpraMgr == nil {
				m.xpraMgr = xpra.New("default", 10)
			}
			m.mu.Unlock()
		}
		if params.Display == "" {
			params.Display = ":10"
		}
		if err := m.xpraMgr.StartServer(params.Display); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "xpra.start.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		info := m.xpraMgr.Info()
		m.sendEvent(sess, Event{Protocol: "ui", Event: "xpra.start.ok", Data: json.RawMessage(mustMarshal(info))})

	case "xpra.stop":
		if m.xpraMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "xpra.stop.error", Data: json.RawMessage(`{"message":"xpra not initialized"}`)})
			return
		}
		if err := m.xpraMgr.StopServer(); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "xpra.stop.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "xpra.stop.ok", Data: json.RawMessage(`{"message":"stopped"}`)})

	case "xpra.attach":
		var params struct {
			Cmd  string   `json:"cmd"`
			Args []string `json:"args"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if m.xpraMgr == nil || !m.xpraMgr.IsRunning() {
			sess.Send(Event{Protocol: "ui", Event: "xpra.attach.error", Data: json.RawMessage(`{"message":"xpra server not running"}`)})
			return
		}
		if params.Cmd == "" {
			sess.Send(Event{Protocol: "ui", Event: "xpra.attach.error", Data: json.RawMessage(`{"message":"cmd is required"}`)})
			return
		}
		ctx := context.Background()
		windowID, err := m.xpraMgr.AttachApp(ctx, params.Cmd, params.Args)
		if err != nil {
			sess.Send(Event{Protocol: "ui", Event: "xpra.attach.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "xpra.attach.ok", Data: json.RawMessage(mustMarshal(map[string]interface{}{"window_id": windowID}))})

	case "xpra.list":
		if m.xpraMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "xpra.list", Data: json.RawMessage(`{"windows":[]}`)})
			return
		}
		windows := m.xpraMgr.ListWindows()
		m.sendEvent(sess, Event{Protocol: "ui", Event: "xpra.list", Data: json.RawMessage(mustMarshal(map[string]interface{}{"windows": windows}))})

	case "xpra.detach":
		var params struct {
			WindowID string `json:"window_id"`
		}
		if env.Params != nil {
			json.Unmarshal(env.Params, &params)
		}
		if m.xpraMgr == nil || params.WindowID == "" {
			sess.Send(Event{Protocol: "ui", Event: "xpra.detach.error", Data: json.RawMessage(`{"message":"invalid request"}`)})
			return
		}
		if err := m.xpraMgr.DetachWindow(params.WindowID); err != nil {
			sess.Send(Event{Protocol: "ui", Event: "xpra.detach.error", Data: json.RawMessage(fmt.Sprintf(`{"message":"%s"}`, err.Error()))})
			return
		}
		m.sendEvent(sess, Event{Protocol: "ui", Event: "xpra.detach.ok", Data: json.RawMessage(mustMarshal(map[string]interface{}{"window_id": params.WindowID}))})

	case "xpra.info":
		if m.xpraMgr == nil {
			sess.Send(Event{Protocol: "ui", Event: "xpra.info", Data: json.RawMessage(`{"display":"","http_url":"","running":false,"num_windows":0}`)})
			return
		}
		info := m.xpraMgr.Info()
		m.sendEvent(sess, Event{Protocol: "ui", Event: "xpra.info", Data: json.RawMessage(mustMarshal(info))})
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

	// ---- Profile handlers ----
	case "profiles.list":
		m.handleProfilesList(sess, env.Params)

	case "profiles.get":
		m.handleProfilesGet(sess, env.Params)

	// ---- Skill handlers ----
	case "skills.list":
		m.handleSkillsList(sess, env.Params)

	case "skills.content":
		m.handleSkillsContent(sess, env.Params)

	// ---- Cron handlers ----
	case "crons.list":
		m.handleCronsList(sess, env.Params)

	case "crons.create":
		m.handleCronsCreate(sess, env.Params)

	case "crons.update":
		m.handleCronsUpdate(sess, env.Params)

	case "crons.delete":
		m.handleCronsDelete(sess, env.Params)

	case "crons.pause":
		m.handleCronsPause(sess, env.Params)

	case "crons.resume":
		m.handleCronsResume(sess, env.Params)

	case "crons.run":
		m.handleCronsRun(sess, env.Params)

	// ---- Memory handlers ----
	case "memory.read":
		m.handleMemoryRead(sess, env.Params)

	case "memory.write":
		m.handleMemoryWrite(sess, env.Params)

	// ---- MCP handlers ----
	case "mcp.list":
		m.handleMCPList(sess, env.Params)

	case "mcp.connect":
		m.handleMCPConnect(sess, env.Params)

	case "mcp.disconnect":
		m.handleMCPDisconnect(sess, env.Params)

	case "mcp.tools.list":
		m.handleMCPToolsList(sess, env.Params)

	case "mcp.tools.call":
		m.handleMCPToolsCall(sess, env.Params)

	case "mcp.resources.list":
		m.handleMCPResourcesList(sess, env.Params)

	case "mcp.resources.read":
		m.handleMCPResourcesRead(sess, env.Params)

	case "mcp.prompts.list":
		m.handleMCPPromptsList(sess, env.Params)

	case "mcp.prompts.get":
		m.handleMCPPromptsGet(sess, env.Params)
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

// handleChatWithHermes streams a chat message to Hermes Agent SSE endpoint
// and forwards tokens/events back to the client via WebSocket.
func (m *Multiplexer) handleChatWithHermes(sess *Session, sessionID string, message string) {
	// Telemetry
	if m.telemetry != nil {
		trunc := message
		if len(trunc) > 100 {
			trunc = trunc[:100]
		}
		m.telemetry.Write(telemetry.Event{SessionID: sessionID, Type: "chat.send", Command: trunc})
	}

	streamID := sessionID
	streamer := agent.NewStreamer(m.hermesURL, "")

	// Send initial streaming-started event
	m.sendEvent(sess, Event{
		Protocol: "agent",
		Event:    "chat.stream_start",
		Data:     json.RawMessage(fmt.Sprintf(`{"session_id":%s}`, mustMarshal(streamID))),
	})

	var buf strings.Builder

	err := streamer.Stream(context.Background(), message, func(evt agent.StreamEvent) {
		switch evt.Type {
		case "token":
			buf.WriteString(evt.Content)
			m.sendEvent(sess, Event{
				Protocol: "agent",
				Event:    "chat.token",
				Data:     json.RawMessage(fmt.Sprintf(`{"content":%s}`, mustMarshal(evt.Content))),
			})
		case "reasoning":
			// Flush any pending text token first
			if buf.Len() > 0 {
				m.sendEvent(sess, Event{
					Protocol: "agent",
					Event:    "chat.token",
					Data:     json.RawMessage(fmt.Sprintf(`{"content":%s}`, mustMarshal(buf.String()))),
				})
				buf.Reset()
			}
			m.sendEvent(sess, Event{
				Protocol: "agent",
				Event:    "chat.reasoning",
				Data:     json.RawMessage(fmt.Sprintf(`{"content":%s}`, mustMarshal(evt.Content))),
			})
		case "tool_call":
			// Flush any pending text
			if buf.Len() > 0 {
				m.sendEvent(sess, Event{
					Protocol: "agent",
					Event:    "chat.token",
					Data:     json.RawMessage(fmt.Sprintf(`{"content":%s}`, mustMarshal(buf.String()))),
				})
				buf.Reset()
			}
			if evt.ToolCall != nil {
				m.sendEvent(sess, Event{
					Protocol: "agent",
					Event:    "chat.tool_call",
					Data:     json.RawMessage(fmt.Sprintf(`{"id":%s,"name":%s,"arguments":%s}`, mustMarshal(evt.ToolCall.ID), mustMarshal(evt.ToolCall.Name), mustMarshal(evt.ToolCall.Args))),
				})
			}
		case "tool_result":
			if buf.Len() > 0 {
				m.sendEvent(sess, Event{
					Protocol: "agent",
					Event:    "chat.token",
					Data:     json.RawMessage(fmt.Sprintf(`{"content":%s}`, mustMarshal(buf.String()))),
				})
				buf.Reset()
			}
			m.sendEvent(sess, Event{
				Protocol: "agent",
				Event:    "chat.tool_result",
				Data:     json.RawMessage(fmt.Sprintf(`{"result":%s}`, mustMarshal(evt.Result))),
			})
		case "stream_end":
			if buf.Len() > 0 {
				m.sendEvent(sess, Event{
					Protocol: "agent",
					Event:    "chat.token",
					Data:     json.RawMessage(fmt.Sprintf(`{"content":%s}`, mustMarshal(buf.String()))),
				})
				buf.Reset()
			}
			m.sendEvent(sess, Event{
				Protocol: "agent",
				Event:    "chat.reply",
				Data:     json.RawMessage(`{"complete":true}`),
			})
		case "error":
			m.sendEvent(sess, Event{
				Protocol: "agent",
				Event:    "chat.error",
				Data:     json.RawMessage(fmt.Sprintf(`{"message":%s}`, mustMarshal(evt.Error))),
			})
		}
	})

	if err != nil {
		log.Printf("hermes streaming error: %v", err)
		m.sendEvent(sess, Event{
			Protocol: "agent",
			Event:    "chat.error",
			Data:     json.RawMessage(fmt.Sprintf(`{"message":%s}`, mustMarshal(err.Error()))),
		})
	}
}

// handleToolExecute streams a tool execution request to the Hermes Agent SSE endpoint
// and sends the result back via WebSocket.
func (m *Multiplexer) handleToolExecute(sess *Session, sessionID string, tileSessionID string, toolName string, args map[string]interface{}) {
	argsJSON, _ := json.Marshal(args)

	hermesURL := m.hermesURL
	if hermesURL == "" {
		hermesURL = m.hermesURL
	}

	streamer := agent.NewStreamer(hermesURL, "")
	toolMsg := fmt.Sprintf("Execute tool %q with arguments: %s", toolName, string(argsJSON))

	var result strings.Builder
	err := streamer.Stream(context.Background(), toolMsg, func(evt agent.StreamEvent) {
		switch evt.Type {
		case "token":
			result.WriteString(evt.Content)
		case "tool_result":
			result.WriteString(evt.Result)
		}
	})
	if err != nil {
		data, _ := json.Marshal(map[string]interface{}{
			"session_id": tileSessionID,
			"tool_name":  toolName,
			"error":      "Hermes agent error: " + err.Error(),
		})
		m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.error", Data: data})
		return
	}

	resultStr := result.String()
	if resultStr == "" {
		resultStr = fmt.Sprintf("Tool %q executed (no output)", toolName)
	}
	resultData := map[string]interface{}{
		"session_id": tileSessionID,
		"tool_name":  toolName,
		"result":     resultStr,
	}
	data, _ := json.Marshal(resultData)
	m.sendEvent(sess, Event{Protocol: "agent", Event: "tool.result", Data: data})
}

// ---- Profile handlers ----

func (m *Multiplexer) handleProfilesList(sess *Session, params json.RawMessage) {
	var p struct {
		Filter string `json:"filter,omitempty"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	profiles := []map[string]interface{}{}
	if m.configMgr != nil {
		cfg := m.configMgr.Get()
		if cfg != nil {
			// Collect personality names, sorted for deterministic output
			names := []string{}
			for name := range cfg.Agent.Personalities {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
				desc := cfg.Agent.Personalities[name]
				profiles = append(profiles, map[string]interface{}{
					"id":          name,
					"name":        name,
					"description": desc,
					"type":        "personality",
					"active":      false,
				})
			}
		}
	}

	m.sendEvent(sess, Event{
		Protocol: "agent",
		Event:    "profiles.list",
		Data:     json.RawMessage(mustMarshal(map[string]interface{}{"profiles": profiles})),
	})
}

func (m *Multiplexer) handleProfilesGet(sess *Session, params json.RawMessage) {
	var p struct {
		ID string `json:"id"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	var profile interface{}
	if m.configMgr != nil && p.ID != "" {
		cfg := m.configMgr.Get()
		if cfg != nil {
			if desc, ok := cfg.Agent.Personalities[p.ID]; ok {
				profile = map[string]interface{}{
					"id":          p.ID,
					"name":        p.ID,
					"description": desc,
					"type":        "personality",
					"active":      false,
				}
			}
		}
	}

	m.sendEvent(sess, Event{
		Protocol: "agent",
		Event:    "profiles.get",
		Data:     json.RawMessage(mustMarshal(map[string]interface{}{"profile": profile})),
	})
}

// ---- Skill handlers ----

func (m *Multiplexer) handleSkillsList(sess *Session, params json.RawMessage) {
	var p struct {
		Category string `json:"category,omitempty"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	type skillEntry struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Enabled     bool   `json:"enabled"`
	}

	skills := []skillEntry{}

	home, err := os.UserHomeDir()
	if err == nil {
		skillsRoot := filepath.Join(home, ".hermes", "skills")

		// Walk <category>/<skill-name>/SKILL.md
		categories, err := os.ReadDir(skillsRoot)
		if err == nil {
			for _, cat := range categories {
				if !cat.IsDir() {
					continue
				}
				catName := cat.Name()
				if p.Category != "" && catName != p.Category {
					continue
				}

				skillDirs, err := os.ReadDir(filepath.Join(skillsRoot, catName))
				if err != nil {
					continue
				}

				for _, skillDir := range skillDirs {
					if !skillDir.IsDir() {
						continue
					}
					skillName := skillDir.Name()
					skillMDPath := filepath.Join(skillsRoot, catName, skillName, "SKILL.md")

					data, err := os.ReadFile(skillMDPath)
					if err != nil {
						continue
					}

					desc := parseSkillDescription(string(data))
					skills = append(skills, skillEntry{
						Name:        skillName,
						Description: desc,
						Category:    catName,
						Enabled:     false,
					})
				}
			}
		}
	}

	// Sort by name for deterministic output
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	m.sendEvent(sess, Event{
		Protocol: "agent",
		Event:    "skills.list",
		Data:     json.RawMessage(mustMarshal(map[string]interface{}{"skills": skills})),
	})
}

func (m *Multiplexer) handleSkillsContent(sess *Session, params json.RawMessage) {
	var p struct {
		Name string `json:"name"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	content := ""

	if p.Name != "" {
		home, err := os.UserHomeDir()
		if err == nil {
			skillsRoot := filepath.Join(home, ".hermes", "skills")

			filepath.Walk(skillsRoot, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // skip inaccessible paths
				}
				if info.IsDir() || filepath.Base(path) != "SKILL.md" {
					return nil
				}

				// Check if this SKILL.md's frontmatter name matches
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				if skillNameMatches(data, p.Name) {
					content = string(data)
					return filepath.SkipAll // found it, stop walking
				}
				return nil
			})
		}
	}

	m.sendEvent(sess, Event{
		Protocol: "agent",
		Event:    "skills.content",
		Data:     json.RawMessage(mustMarshal(map[string]interface{}{"content": content})),
	})
}

// parseSkillDescription extracts the description field from YAML frontmatter in a SKILL.md file.
// It uses simple line parsing rather than a YAML library.
func parseSkillDescription(data string) string {
	// Look for YAML frontmatter between --- markers
	trimmed := strings.TrimSpace(data)
	if !strings.HasPrefix(trimmed, "---") {
		return ""
	}

	// Find end of frontmatter
	rest := trimmed[3:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return ""
	}

	frontmatter := rest[:endIdx]
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			val := strings.TrimSpace(line[len("description:"):])
			// Remove surrounding quotes if present
			val = strings.Trim(val, `"`)
			val = strings.Trim(val, `'`)
			return val
		}
	}
	return ""
}

// skillNameMatches checks if the SKILL.md file's frontmatter name matches the given name.
func skillNameMatches(data []byte, name string) bool {
	content := string(data)
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return filepath.Base(string(data)) == name // fallback to filename
	}

	rest := trimmed[3:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return false
	}

	frontmatter := rest[:endIdx]
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			val := strings.TrimSpace(line[len("name:"):])
			val = strings.Trim(val, `"`)
			val = strings.Trim(val, `'`)
			return val == name
		}
	}
	return false
}

// ---- Cron handlers ----

func (m *Multiplexer) handleCronsList(sess *Session, params json.RawMessage) {
	home, err := os.UserHomeDir()
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.list", Data: json.RawMessage(`{"crons":[]}`)})
		return
	}
	path := filepath.Join(home, ".hermes", "cron", "jobs.json")
	data, err := os.ReadFile(path)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.list", Data: json.RawMessage(`{"crons":[]}`)})
		return
	}
	var result struct {
		Jobs []json.RawMessage `json:"jobs"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.list", Data: json.RawMessage(`{"crons":[]}`)})
		return
	}
	if result.Jobs == nil {
		result.Jobs = []json.RawMessage{}
	}
	m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.list", Data: json.RawMessage(mustMarshal(map[string]interface{}{"crons": result.Jobs}))})
}

func (m *Multiplexer) handleCronsCreate(sess *Session, params json.RawMessage) {
	var p struct {
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
		Command  string `json:"command"`
		Enabled  bool   `json:"enabled"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.create", Data: json.RawMessage(`{"cron":{"id":""}}`)})
		return
	}
	cronPath := filepath.Join(home, ".hermes", "cron", "jobs.json")

	// Read existing jobs
	var cronFile struct {
		Jobs      []json.RawMessage `json:"jobs"`
		UpdatedAt string            `json:"updated_at"`
	}
	data, err := os.ReadFile(cronPath)
	if err == nil {
		json.Unmarshal(data, &cronFile)
	}
	if cronFile.Jobs == nil {
		cronFile.Jobs = []json.RawMessage{}
	}

	// Generate unique 12-char hex ID
	id := fmt.Sprintf("%x", rand.Int63())

	// Build schedule display
	scheduleDisplay := p.Schedule
	if scheduleDisplay == "" {
		scheduleDisplay = "once"
	}

	// Create new job map
	newJob := map[string]interface{}{
		"id":              id,
		"name":            p.Name,
		"prompt":          p.Command,
		"schedule":        map[string]interface{}{"kind": "cron", "cron": p.Schedule, "display": scheduleDisplay},
		"schedule_display": scheduleDisplay,
		"enabled":         p.Enabled,
		"state":           "scheduled",
		"created_at":      time.Now().UTC().Format(time.RFC3339Nano),
		"last_run_at":     nil,
		"last_status":     nil,
		"last_error":      nil,
		"model":           nil,
		"provider":        nil,
	}

	jobData, _ := json.Marshal(newJob)
	cronFile.Jobs = append(cronFile.Jobs, jobData)
	cronFile.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)

	outData, _ := json.Marshal(cronFile)
	os.WriteFile(cronPath, outData, 0644)

	m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.create", Data: json.RawMessage(mustMarshal(map[string]interface{}{"cron": map[string]string{"id": id}}))})
}

func (m *Multiplexer) handleCronsUpdate(sess *Session, params json.RawMessage) {
	var p struct {
		ID       string `json:"id"`
		Name     string `json:"name,omitempty"`
		Schedule string `json:"schedule,omitempty"`
		Command  string `json:"command,omitempty"`
		Enabled  *bool  `json:"enabled,omitempty"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.update", Data: json.RawMessage(`{"success":false,"error":"cannot determine home directory"}`)})
		return
	}
	cronPath := filepath.Join(home, ".hermes", "cron", "jobs.json")

	data, err := os.ReadFile(cronPath)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.update", Data: json.RawMessage(`{"success":false,"error":"cannot read cron file"}`)})
		return
	}

	var cronFile struct {
		Jobs      []json.RawMessage `json:"jobs"`
		UpdatedAt string            `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &cronFile); err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.update", Data: json.RawMessage(`{"success":false,"error":"invalid cron file"}`)})
		return
	}

	found := false
	for i, jobData := range cronFile.Jobs {
		var job map[string]interface{}
		if err := json.Unmarshal(jobData, &job); err != nil {
			continue
		}
		jobID, _ := job["id"].(string)
		if jobID != p.ID {
			continue
		}
		found = true

		if p.Name != "" {
			job["name"] = p.Name
		}
		if p.Command != "" {
			job["prompt"] = p.Command
		}
		if p.Schedule != "" {
			job["schedule"] = map[string]interface{}{"kind": "cron", "cron": p.Schedule, "display": p.Schedule}
			job["schedule_display"] = p.Schedule
		}
		if p.Enabled != nil {
			job["enabled"] = *p.Enabled
		}

		updatedData, _ := json.Marshal(job)
		cronFile.Jobs[i] = updatedData
		break
	}

	if !found {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.update", Data: json.RawMessage(`{"success":false,"error":"job not found"}`)})
		return
	}

	cronFile.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	outData, _ := json.Marshal(cronFile)
	os.WriteFile(cronPath, outData, 0644)

	m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.update", Data: json.RawMessage(`{"success":true}`)})
}

func (m *Multiplexer) handleCronsDelete(sess *Session, params json.RawMessage) {
	var p struct {
		ID string `json:"id"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.delete", Data: json.RawMessage(`{"success":false,"error":"cannot determine home directory"}`)})
		return
	}
	cronPath := filepath.Join(home, ".hermes", "cron", "jobs.json")

	data, err := os.ReadFile(cronPath)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.delete", Data: json.RawMessage(`{"success":false,"error":"cannot read cron file"}`)})
		return
	}

	var cronFile struct {
		Jobs      []json.RawMessage `json:"jobs"`
		UpdatedAt string            `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &cronFile); err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.delete", Data: json.RawMessage(`{"success":false,"error":"invalid cron file"}`)})
		return
	}

	found := false
	filtered := make([]json.RawMessage, 0, len(cronFile.Jobs))
	for _, jobData := range cronFile.Jobs {
		var job map[string]interface{}
		if err := json.Unmarshal(jobData, &job); err != nil {
			continue
		}
		jobID, _ := job["id"].(string)
		if jobID == p.ID {
			found = true
			continue // skip this job (delete it)
		}
		filtered = append(filtered, jobData)
	}

	if !found {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.delete", Data: json.RawMessage(`{"success":false,"error":"job not found"}`)})
		return
	}

	cronFile.Jobs = filtered
	cronFile.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	outData, _ := json.Marshal(cronFile)
	os.WriteFile(cronPath, outData, 0644)

	m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.delete", Data: json.RawMessage(`{"success":true}`)})
}

func (m *Multiplexer) handleCronsPause(sess *Session, params json.RawMessage) {
	var p struct {
		ID string `json:"id"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.pause", Data: json.RawMessage(`{"success":false,"error":"cannot determine home directory"}`)})
		return
	}
	cronPath := filepath.Join(home, ".hermes", "cron", "jobs.json")

	data, err := os.ReadFile(cronPath)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.pause", Data: json.RawMessage(`{"success":false,"error":"cannot read cron file"}`)})
		return
	}

	var cronFile struct {
		Jobs      []json.RawMessage `json:"jobs"`
		UpdatedAt string            `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &cronFile); err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.pause", Data: json.RawMessage(`{"success":false,"error":"invalid cron file"}`)})
		return
	}

	found := false
	for i, jobData := range cronFile.Jobs {
		var job map[string]interface{}
		if err := json.Unmarshal(jobData, &job); err != nil {
			continue
		}
		jobID, _ := job["id"].(string)
		if jobID != p.ID {
			continue
		}
		found = true
		job["state"] = "paused"
		job["enabled"] = false
		now := time.Now().UTC().Format(time.RFC3339Nano)
		job["paused_at"] = now
		updatedData, _ := json.Marshal(job)
		cronFile.Jobs[i] = updatedData
		break
	}

	if !found {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.pause", Data: json.RawMessage(`{"success":false,"error":"job not found"}`)})
		return
	}

	cronFile.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	outData, _ := json.Marshal(cronFile)
	os.WriteFile(cronPath, outData, 0644)

	m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.pause", Data: json.RawMessage(`{"success":true}`)})
}

func (m *Multiplexer) handleCronsResume(sess *Session, params json.RawMessage) {
	var p struct {
		ID string `json:"id"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.resume", Data: json.RawMessage(`{"success":false,"error":"cannot determine home directory"}`)})
		return
	}
	cronPath := filepath.Join(home, ".hermes", "cron", "jobs.json")

	data, err := os.ReadFile(cronPath)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.resume", Data: json.RawMessage(`{"success":false,"error":"cannot read cron file"}`)})
		return
	}

	var cronFile struct {
		Jobs      []json.RawMessage `json:"jobs"`
		UpdatedAt string            `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &cronFile); err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.resume", Data: json.RawMessage(`{"success":false,"error":"invalid cron file"}`)})
		return
	}

	found := false
	for i, jobData := range cronFile.Jobs {
		var job map[string]interface{}
		if err := json.Unmarshal(jobData, &job); err != nil {
			continue
		}
		jobID, _ := job["id"].(string)
		if jobID != p.ID {
			continue
		}
		found = true
		job["state"] = "scheduled"
		job["enabled"] = true
		updatedData, _ := json.Marshal(job)
		cronFile.Jobs[i] = updatedData
		break
	}

	if !found {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.resume", Data: json.RawMessage(`{"success":false,"error":"job not found"}`)})
		return
	}

	cronFile.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	outData, _ := json.Marshal(cronFile)
	os.WriteFile(cronPath, outData, 0644)

	m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.resume", Data: json.RawMessage(`{"success":true}`)})
}

func (m *Multiplexer) handleCronsRun(sess *Session, params json.RawMessage) {
	var p struct {
		ID string `json:"id"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.run", Data: json.RawMessage(`{"success":false,"error":"cannot determine home directory"}`)})
		return
	}
	cronPath := filepath.Join(home, ".hermes", "cron", "jobs.json")

	data, err := os.ReadFile(cronPath)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.run", Data: json.RawMessage(`{"success":false,"error":"cannot read cron file"}`)})
		return
	}

	var cronFile struct {
		Jobs      []json.RawMessage `json:"jobs"`
		UpdatedAt string            `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &cronFile); err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.run", Data: json.RawMessage(`{"success":false,"error":"invalid cron file"}`)})
		return
	}

	found := false
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for i, jobData := range cronFile.Jobs {
		var job map[string]interface{}
		if err := json.Unmarshal(jobData, &job); err != nil {
			continue
		}
		jobID, _ := job["id"].(string)
		if jobID != p.ID {
			continue
		}
		found = true
		job["last_run_at"] = now
		job["state"] = "running"
		updatedData, _ := json.Marshal(job)
		cronFile.Jobs[i] = updatedData
		break
	}

	if !found {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.run", Data: json.RawMessage(`{"success":false,"error":"job not found"}`)})
		return
	}

	cronFile.UpdatedAt = now
	outData, _ := json.Marshal(cronFile)
	os.WriteFile(cronPath, outData, 0644)

	m.sendEvent(sess, Event{Protocol: "agent", Event: "crons.run", Data: json.RawMessage(`{"success":true}`)})
}

// ---- Memory handlers ----

func (m *Multiplexer) handleMemoryRead(sess *Session, params json.RawMessage) {
	var p struct {
		Namespace string `json:"namespace,omitempty"`
		Key       string `json:"key,omitempty"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "memory.read", Data: json.RawMessage(`{"value":""}`)})
		return
	}

	// Determine which memory file to read
	namespace := p.Namespace
	if namespace == "" {
		namespace = "memory"
	}

	var fileName string
	if namespace == "user" {
		fileName = "USER.md"
	} else {
		fileName = "MEMORY.md"
	}

	memPath := filepath.Join(home, ".hermes", "memories", fileName)
	data, err := os.ReadFile(memPath)
	if err != nil {
		// File doesn't exist or can't be read — return empty
		m.sendEvent(sess, Event{Protocol: "agent", Event: "memory.read", Data: json.RawMessage(`{"value":""}`)})
		return
	}

	content := string(data)

	// If key is provided, search for that key in the content
	if p.Key != "" {
		lines := strings.Split(content, "\n")
		var matched strings.Builder
		keyPrefix := strings.ToLower(p.Key) + ":"
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(trimmed), keyPrefix) {
				if matched.Len() > 0 {
					matched.WriteString("\n")
				}
				matched.WriteString(line)
			}
		}
		content = matched.String()
	}

	m.sendEvent(sess, Event{Protocol: "agent", Event: "memory.read", Data: json.RawMessage(mustMarshal(map[string]string{"value": content}))})
}

func (m *Multiplexer) handleMemoryWrite(sess *Session, params json.RawMessage) {
	var p struct {
		Namespace string `json:"namespace,omitempty"`
		Key       string `json:"key"`
		Value     string `json:"value"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "memory.write", Data: json.RawMessage(`{"success":false,"error":"cannot determine home directory"}`)})
		return
	}

	namespace := p.Namespace
	if namespace == "" {
		namespace = "memory"
	}

	var fileName string
	if namespace == "user" {
		fileName = "USER.md"
	} else {
		fileName = "MEMORY.md"
	}

	memPath := filepath.Join(home, ".hermes", "memories", fileName)

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(memPath), 0755)

	if p.Key != "" {
		// Read existing content and update the specific key
		existingContent := ""
		if data, err := os.ReadFile(memPath); err == nil {
			existingContent = string(data)
		}

		lines := strings.Split(existingContent, "\n")
		keyPrefix := strings.ToLower(p.Key) + ":"
		found := false
		var updated strings.Builder

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(trimmed), keyPrefix) {
				// Replace this line with the new key: value
				if updated.Len() > 0 {
					updated.WriteString("\n")
				}
				updated.WriteString(p.Key + ": " + p.Value)
				found = true
			} else {
				if updated.Len() > 0 {
					updated.WriteString("\n")
				}
				updated.WriteString(line)
			}
		}

		if !found {
			// Append new key: value line
			if updated.Len() > 0 {
				updated.WriteString("\n")
			}
			updated.WriteString(p.Key + ": " + p.Value)
		}

		os.WriteFile(memPath, []byte(updated.String()), 0644)
	} else {
		// No key — write the value directly as the entire file content
		os.WriteFile(memPath, []byte(p.Value), 0644)
	}

	m.sendEvent(sess, Event{Protocol: "agent", Event: "memory.write", Data: json.RawMessage(`{"success":true}`)})
}

// ---- MCP handlers ----

func (m *Multiplexer) handleMCPList(sess *Session, params json.RawMessage) {
	if m.mcpMgr == nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.list", Data: json.RawMessage(`{"servers":[],"error":"mcp manager not available"}`)})
		return
	}
	clients := m.mcpMgr.ListClients()
	m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.list", Data: json.RawMessage(mustMarshal(map[string]interface{}{"servers": clients}))})
}

func (m *Multiplexer) handleMCPConnect(sess *Session, params json.RawMessage) {
	var p struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args,omitempty"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if p.Name == "" || p.Command == "" {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.connect", Data: json.RawMessage(`{"success":false,"error":"name and command are required"}`)})
		return
	}
	if m.mcpMgr == nil {
		m.mcpMgr = mcp.NewManager()
	}
	client := m.mcpMgr.AddClient(p.Name, p.Command, p.Args...)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.Start(ctx); err != nil {
		m.mcpMgr.RemoveClient(p.Name)
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.connect", Data: json.RawMessage(fmt.Sprintf(`{"success":false,"error":"%s"}`, err.Error()))})
		return
	}
	if _, err := client.Initialize(ctx); err != nil {
		client.Stop()
		m.mcpMgr.RemoveClient(p.Name)
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.connect", Data: json.RawMessage(fmt.Sprintf(`{"success":false,"error":"%s"}`, err.Error()))})
		return
	}
	m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.connect", Data: json.RawMessage(mustMarshal(map[string]interface{}{"success": true, "name": p.Name}))})
}

func (m *Multiplexer) handleMCPDisconnect(sess *Session, params json.RawMessage) {
	var p struct {
		Name string `json:"name"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if m.mcpMgr == nil || p.Name == "" {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.disconnect", Data: json.RawMessage(`{"success":false,"error":"client not found"}`)})
		return
	}
	if err := m.mcpMgr.RemoveClient(p.Name); err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.disconnect", Data: json.RawMessage(fmt.Sprintf(`{"success":false,"error":"%s"}`, err.Error()))})
		return
	}
	m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.disconnect", Data: json.RawMessage(mustMarshal(map[string]interface{}{"success": true, "name": p.Name}))})
}

func (m *Multiplexer) handleMCPToolsList(sess *Session, params json.RawMessage) {
	var p struct {
		ServerName string `json:"server_name"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if m.mcpMgr == nil || p.ServerName == "" {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.tools.list", Data: json.RawMessage(`{"tools":[],"error":"invalid request"}`)})
		return
	}
	client, ok := m.mcpMgr.GetClient(p.ServerName)
	if !ok {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.tools.list", Data: json.RawMessage(`{"tools":[],"error":"client not found"}`)})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tools, err := client.ListTools(ctx)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.tools.list", Data: json.RawMessage(fmt.Sprintf(`{"tools":[],"error":"%s"}`, err.Error()))})
		return
	}
	m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.tools.list", Data: json.RawMessage(mustMarshal(map[string]interface{}{"server": p.ServerName, "tools": tools}))})
}

func (m *Multiplexer) handleMCPToolsCall(sess *Session, params json.RawMessage) {
	var p struct {
		ServerName string                 `json:"server_name"`
		ToolName   string                 `json:"tool_name"`
		Arguments  map[string]interface{} `json:"arguments,omitempty"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if m.mcpMgr == nil || p.ServerName == "" || p.ToolName == "" {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.tools.call", Data: json.RawMessage(`{"error":"invalid request"}`)})
		return
	}
	client, ok := m.mcpMgr.GetClient(p.ServerName)
	if !ok {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.tools.call", Data: json.RawMessage(`{"error":"client not found"}`)})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, err := client.CallTool(ctx, p.ToolName, p.Arguments)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.tools.call", Data: json.RawMessage(fmt.Sprintf(`{"error":"%s"}`, err.Error()))})
		return
	}
	m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.tools.call", Data: json.RawMessage(mustMarshal(map[string]interface{}{"success": true, "server": p.ServerName, "tool": p.ToolName, "result": result}))})
}

func (m *Multiplexer) handleMCPResourcesList(sess *Session, params json.RawMessage) {
	var p struct {
		ServerName string `json:"server_name"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if m.mcpMgr == nil || p.ServerName == "" {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.resources.list", Data: json.RawMessage(`{"resources":[],"error":"invalid request"}`)})
		return
	}
	client, ok := m.mcpMgr.GetClient(p.ServerName)
	if !ok {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.resources.list", Data: json.RawMessage(`{"resources":[],"error":"client not found"}`)})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resources, err := client.ListResources(ctx)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.resources.list", Data: json.RawMessage(fmt.Sprintf(`{"resources":[],"error":"%s"}`, err.Error()))})
		return
	}
	m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.resources.list", Data: json.RawMessage(mustMarshal(map[string]interface{}{"server": p.ServerName, "resources": resources}))})
}

func (m *Multiplexer) handleMCPResourcesRead(sess *Session, params json.RawMessage) {
	var p struct {
		ServerName string `json:"server_name"`
		URI        string `json:"uri"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if m.mcpMgr == nil || p.ServerName == "" || p.URI == "" {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.resources.read", Data: json.RawMessage(`{"error":"invalid request"}`)})
		return
	}
	client, ok := m.mcpMgr.GetClient(p.ServerName)
	if !ok {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.resources.read", Data: json.RawMessage(`{"error":"client not found"}`)})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := client.ReadResource(ctx, p.URI)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.resources.read", Data: json.RawMessage(fmt.Sprintf(`{"error":"%s"}`, err.Error()))})
		return
	}
	m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.resources.read", Data: json.RawMessage(mustMarshal(map[string]interface{}{"success": true, "server": p.ServerName, "uri": p.URI, "result": result}))})
}

func (m *Multiplexer) handleMCPPromptsList(sess *Session, params json.RawMessage) {
	var p struct {
		ServerName string `json:"server_name"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if m.mcpMgr == nil || p.ServerName == "" {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.prompts.list", Data: json.RawMessage(`{"prompts":[],"error":"invalid request"}`)})
		return
	}
	client, ok := m.mcpMgr.GetClient(p.ServerName)
	if !ok {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.prompts.list", Data: json.RawMessage(`{"prompts":[],"error":"client not found"}`)})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	prompts, err := client.ListPrompts(ctx)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.prompts.list", Data: json.RawMessage(fmt.Sprintf(`{"prompts":[],"error":"%s"}`, err.Error()))})
		return
	}
	m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.prompts.list", Data: json.RawMessage(mustMarshal(map[string]interface{}{"server": p.ServerName, "prompts": prompts}))})
}

func (m *Multiplexer) handleMCPPromptsGet(sess *Session, params json.RawMessage) {
	var p struct {
		ServerName string                 `json:"server_name"`
		Name       string                 `json:"name"`
		Arguments  map[string]interface{} `json:"arguments,omitempty"`
	}
	if params != nil {
		json.Unmarshal(params, &p)
	}
	if m.mcpMgr == nil || p.ServerName == "" || p.Name == "" {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.prompts.get", Data: json.RawMessage(`{"error":"invalid request"}`)})
		return
	}
	client, ok := m.mcpMgr.GetClient(p.ServerName)
	if !ok {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.prompts.get", Data: json.RawMessage(`{"error":"client not found"}`)})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := client.GetPrompt(ctx, p.Name, p.Arguments)
	if err != nil {
		m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.prompts.get", Data: json.RawMessage(fmt.Sprintf(`{"error":"%s"}`, err.Error()))})
		return
	}
	m.sendEvent(sess, Event{Protocol: "agent", Event: "mcp.prompts.get", Data: json.RawMessage(mustMarshal(map[string]interface{}{"success": true, "server": p.ServerName, "name": p.Name, "result": result}))})
}

// getTotalMemGB returns total system memory in GB from /proc/meminfo.
func getTotalMemGB() float64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kb, _ := strconv.ParseInt(fields[1], 10, 64)
			return float64(kb) / 1024 / 1024
		}
	}
	return 0
}

// readHostMemInfo returns used and total memory in GB from /proc/meminfo.
func readHostMemInfo() (used, total float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var memTotal, memFree, buffers, cached int64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			memTotal, _ = strconv.ParseInt(fields[1], 10, 64)
		case "MemFree:":
			memFree, _ = strconv.ParseInt(fields[1], 10, 64)
		case "Buffers:":
			buffers, _ = strconv.ParseInt(fields[1], 10, 64)
		case "Cached:":
			cached, _ = strconv.ParseInt(fields[1], 10, 64)
		}
	}
	totalGB := float64(memTotal) / 1024 / 1024
	usedGB := float64(memTotal-memFree-buffers-cached) / 1024 / 1024
	return usedGB, totalGB
}

// getCPUUsage returns overall CPU usage percentage (0-100) from /proc/stat.
func getCPUUsage() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		var total, idle uint64
		for i := 1; i < len(fields); i++ {
			v, _ := strconv.ParseUint(fields[i], 10, 64)
			total += v
			if i == 4 {
				idle = v
			}
		}
		if total > 0 {
			return (1.0 - float64(idle)/float64(total)) * 100
		}
	}
	return 0
}

// getDiskUsage returns used disk space in GB for the root filesystem.
func getDiskUsage() float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0
	}
	used := (int64(stat.Blocks) - int64(stat.Bfree)) * int64(stat.Bsize)
	return float64(used) / 1024 / 1024 / 1024
}

// getDiskTotal returns total disk space in GB for the root filesystem.
func getDiskTotal() float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
	 return 0
	}
	total := int64(stat.Blocks) * int64(stat.Bsize)
	return float64(total) / 1024 / 1024 / 1024
}

// getDiskPercent returns disk usage percentage (0-100) for the root filesystem.
func getDiskPercent() float64 {
	used := getDiskUsage()
	total := getDiskTotal()
	if total > 0 {
		return (used / total) * 100
	}
	return 0
}

