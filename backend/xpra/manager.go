package xpra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// WindowInfo describes a window managed by Xpra.
type WindowInfo struct {
	WindowID string `json:"window_id"`
	PID      int    `json:"pid"`
	Title    string `json:"title"`
	Geometry string `json:"geometry"`
}

// Manager handles the per-session Xpra server lifecycle.
type Manager struct {
	sessionID  string
	displayNum int
	sockPath   string
	httpPort   int
	cmd        *exec.Cmd
	mu         sync.Mutex
	started    bool
	windows    map[string]WindowInfo // windowID -> info
	windowSeq  int
}

// New creates a Manager for the given session.
func New(sessionID string, displayNum int) *Manager {
	uid := os.Geteuid()
	sockDir := fmt.Sprintf("/run/user/%d/xpra", uid)
	sockPath := filepath.Join(sockDir, fmt.Sprintf("%d", displayNum))
	httpPort := 9443 + displayNum
	return &Manager{
		sessionID:  sessionID,
		displayNum: displayNum,
		sockPath:   sockPath,
		httpPort:   httpPort,
		windows:    make(map[string]WindowInfo),
		windowSeq:  0,
	}
}

// Display returns the X display string (e.g., ":10").
func (m *Manager) Display() string {
	return fmt.Sprintf(":%d", m.displayNum)
}

// HTTPURL returns the URL for the Xpra HTML5 client.
func (m *Manager) HTTPURL() string {
	return fmt.Sprintf("http://localhost:%d", m.httpPort)
}

// HTTPPort returns the HTTP port for the Xpra HTML5 client.
func (m *Manager) HTTPPort() int {
	return m.httpPort
}

// SockPath returns the unix socket path for this session.
func (m *Manager) SockPath() string {
	return m.sockPath
}

// StartServer launches the Xpra server with Xvfb.
// display parameter lets caller specify which display number to use (e.g., ":10").
func (m *Manager) StartServer(display string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return nil
	}

	// Parse display number from string like ":10"
	if display != "" {
		var n int
		if _, err := fmt.Sscanf(display, ":%d", &n); err == nil {
			m.displayNum = n
			// Update sockPath and httpPort for new display
			uid := os.Geteuid()
			sockDir := fmt.Sprintf("/run/user/%d/xpra", uid)
			m.sockPath = filepath.Join(sockDir, fmt.Sprintf("%d", m.displayNum))
			m.httpPort = 9443 + m.displayNum
		}
	}

	sockDir := filepath.Dir(m.sockPath)
	if err := os.MkdirAll(sockDir, 0755); err != nil {
		return fmt.Errorf("create xpra socket dir: %w", err)
	}

	// Check if Xpra is available
	if err := checkXpra(); err != nil {
		return fmt.Errorf("xpra not available: %w", err)
	}

	// Build xpra start arguments
	// Use mode "start" for headless with Xvfb, HTML5 enabled
	args := []string{
		"start",
		"--daemon",
		"--no-daemonize",
		fmt.Sprintf("--socket-dir=%s", sockDir),
		display, // the display itself, e.g. ":10"
		"--bind=TCP",
		fmt.Sprintf("--bind-web=%d", m.httpPort),
		"--html=on", // enable HTML5 client
		"--idle-timeout=0",
		"--server-idle-timeout=0",
		"--exit-with-children",
		"--start=xterm",
		"--encoding=auto",
		"--quality=50",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "xpra", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("xpra start: %w", err)
	}

	m.cmd = cmd
	m.started = true

	// Wait for socket to appear (Xpra server to be ready)
	if err := m.waitForSocket(15 * time.Second); err != nil {
		// Try to clean up
		m.cmd.Process.Kill()
		m.started = false
		return fmt.Errorf("xpra socket not created within 15s: %w", err)
	}

	// Wait for HTTP server to be ready
	if err := m.waitForHTTP(10 * time.Second); err != nil {
		return fmt.Errorf("xpra HTTP server not ready: %w", err)
	}

	return nil
}

func checkXpra() error {
	cmd := exec.Command("xpra", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xpra not installed: %w", err)
	}
	if len(output) > 0 {
		fmt.Printf("[xpra] version: %s", output)
	}
	return nil
}

func (m *Manager) waitForSocket(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(m.sockPath); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("xpra socket not created within %v (path: %s)", timeout, m.sockPath)
}

func (m *Manager) waitForHTTP(timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(m.HTTPURL())
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("xpra HTTP server not ready within %v", timeout)
}

// StopServer terminates the Xpra server cleanly.
func (m *Manager) StopServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	// Send stop signal via xpra command
	stopCmd := exec.Command("xpra", "stop", m.Display())
	stopCmd.Run()

	// Kill the Xpra process if still running
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
		m.cmd = nil
	}

	m.started = false
	m.windows = make(map[string]WindowInfo)
	return nil
}

// AttachApp launches an application and attaches its windows to the Xpra session.
// cmd is the executable path, args are the command-line arguments.
func (m *Manager) AttachApp(ctx context.Context, cmd string, args []string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return "", fmt.Errorf("xpra server not running")
	}

	// Build environment with DISPLAY set to our Xvfb display
	env := os.Environ()
	env = append(env, fmt.Sprintf("DISPLAY=%s", m.Display()))

	execCmd := exec.CommandContext(ctx, cmd, args...)
	execCmd.Env = env
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Start(); err != nil {
		return "", fmt.Errorf("attach app %s: %w", cmd, err)
	}

	// Generate a window ID for tracking
	m.windowSeq++
	windowID := fmt.Sprintf("win_%d_%d", m.displayNum, m.windowSeq)

	// Store window info
	pid := execCmd.Process.Pid
	title := filepath.Base(cmd)
	m.windows[windowID] = WindowInfo{
		WindowID: windowID,
		PID:      pid,
		Title:    title,
	}

	go func() {
		execCmd.Wait()
		// Remove window when process exits
		m.mu.Lock()
		delete(m.windows, windowID)
		m.mu.Unlock()
	}()

	return windowID, nil
}

// ListWindows returns the list of windows currently attached to this session.
func (m *Manager) ListWindows() []WindowInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	windows := make([]WindowInfo, 0, len(m.windows))
	for _, w := range m.windows {
		windows = append(windows, w)
	}
	return windows
}

// DetachWindow closes a window by its ID.
// Uses xpra's control interface to close the specific window.
func (m *Manager) DetachWindow(windowID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, ok := m.windows[windowID]
	if !ok {
		return fmt.Errorf("window not found: %s", windowID)
	}

	// Use xpra control to close the window by PID
	detachCmd := exec.Command("xpra", "control", m.Display(),
		fmt.Sprintf("close window pid=%d", win.PID))
	detachCmd.Run()

	delete(m.windows, windowID)
	return nil
}

// IsRunning returns true if the Xpra server is running.
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

// Start launches the Xpra server on display :10.
// Provided for backward compatibility with apps.go.
func (m *Manager) Start(ctx context.Context) error {
	return m.StartServer(":10")
}

// AttachWindow runs an application and attaches its windows to the Xpra server.
// Deprecated: Use AttachApp instead.
func (m *Manager) AttachWindow(ctx context.Context, app string, args []string) (*exec.Cmd, error) {
	_, err := m.AttachApp(ctx, app, args)
	if err != nil {
		return nil, err
	}
	return nil, nil // we don't track the exec.Cmd anymore
}

// ServerInfo returns information about the Xpra server.
type ServerInfo struct {
	Display    string `json:"display"`
	HTTPURL    string `json:"http_url"`
	Running    bool   `json:"running"`
	NumWindows int    `json:"num_windows"`
}

// Info returns server information.
func (m *Manager) Info() ServerInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return ServerInfo{
		Display:    m.Display(),
		HTTPURL:    m.HTTPURL(),
		Running:    m.started,
		NumWindows: len(m.windows),
	}
}

// MarshalJSON implements json.Marshaler for WindowInfo.
func (w WindowInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(w)
}