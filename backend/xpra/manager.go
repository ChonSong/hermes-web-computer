package xpra

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles the per-session Xpra server lifecycle.
type Manager struct {
	sessionID   string
	displayNum  int
	sockPath    string
	httpPort    int
	cmd         *exec.Cmd
	mu          sync.Mutex
	started     bool
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

// Start launches the Xpra server with Xvfb.
// Uses mode "server" (no root window), starts html5 web bridge.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return nil
	}

	sockDir := filepath.Dir(m.sockPath)
	if err := os.MkdirAll(sockDir, 0755); err != nil {
		return fmt.Errorf("create xpra socket dir: %w", err)
	}

	// Use Xvfb for headless operation
	xvfbDisplay := fmt.Sprintf(":%d", m.displayNum)

	// Check if Xpra is available
	if err := checkXpra(); err != nil {
		return fmt.Errorf("xpra not available: %w", err)
	}

	args := []string{
		"start",
		"--daemon",
		"--no-daemonize",
		fmt.Sprintf("--socket-dir=%s", sockDir),
		"--xvfb=" + xvfbDisplay,
		m.Display(),
		"--bind=TCP",
		fmt.Sprintf("--bind-web=%d", m.httpPort),
		"--html=on",
		"--idle-timeout=0",
		"--server-idle-timeout=0",
		"--exit-with-children",
		"--start=xterm",
		"--encoding=auto",
		"--quality=50",
	}

	cmd := exec.CommandContext(ctx, "xpra", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("xpra start: %w", err)
	}

	m.cmd = cmd
	m.started = true

	// Wait for socket to appear
	return m.waitForSocket(10 * time.Second)
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

// Stop terminates the Xpra server.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
	}
	m.started = false
	exec.Command("xpra", "stop", m.Display()).Run()
	return nil
}

// AttachWindow runs an application and attaches its windows to the Xpra server.
func (m *Manager) AttachWindow(ctx context.Context, app string, args []string) (*exec.Cmd, error) {
	env := os.Environ()
	env = append(env, fmt.Sprintf("DISPLAY=%s", m.Display()))

	cmd := exec.CommandContext(ctx, app, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("attach window %s: %w", app, err)
	}
	return cmd, nil
}

// IsRunning returns true if the Xpra server is running.
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}