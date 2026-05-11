package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// Instance represents a single browser context.
type Instance struct {
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	url        string
	screenshot []byte
}

// Manager tracks browser instances by session ID.
type Manager struct {
	mu        sync.RWMutex
	instances map[string]*Instance
}

// NewManager creates a new browser manager.
func NewManager() *Manager {
	return &Manager{
		instances: make(map[string]*Instance),
	}
}

// Launch creates a new browser instance for the given session.
func (m *Manager) Launch(sessionID string) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.WindowSize(1280, 900),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)

	if err := chromedp.Run(ctx); err != nil {
		allocCancel()
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	inst := &Instance{
		ctx:    ctx,
		cancel: func() { cancel(); allocCancel() },
		url:    "about:blank",
	}
	m.instances[sessionID] = inst
	return inst, nil
}

// GetInstance returns the browser instance for a session.
func (m *Manager) GetInstance(sessionID string) *Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.instances[sessionID]
}

// Close shuts down a browser instance.
func (m *Manager) Close(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inst, ok := m.instances[sessionID]; ok {
		inst.cancel()
		delete(m.instances, sessionID)
	}
}

// Navigate navigates the browser to the given URL.
func (inst *Instance) Navigate(url string) error {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := context.WithTimeout(inst.ctx, 30*time.Second)
	defer cancel()

	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(500), // Let page render
	); err != nil {
		return fmt.Errorf("navigate failed: %w", err)
	}

	inst.url = url
	return nil
}

// Screenshot takes a screenshot and returns base64-encoded PNG.
func (inst *Instance) Screenshot() (string, error) {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := context.WithTimeout(inst.ctx, 15*time.Second)
	defer cancel()

	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.FullScreenshot(&buf, 90),
	); err != nil {
		return "", fmt.Errorf("screenshot failed: %w", err)
	}

	inst.screenshot = buf
	return base64.StdEncoding.EncodeToString(buf), nil
}

// Click simulates a mouse click at the given (x, y) coordinates.
func (inst *Instance) Click(x, y float64) error {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := context.WithTimeout(inst.ctx, 10*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		input.DispatchMouseEvent(input.MousePressed, x, y).
			WithButton(input.Left).
			WithClickCount(1),
		input.DispatchMouseEvent(input.MouseReleased, x, y).
			WithButton(input.Left).
			WithClickCount(1),
	)
}

// Input sends text to the focused element.
func (inst *Instance) Input(text string) error {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := context.WithTimeout(inst.ctx, 10*time.Second)
	defer cancel()

	// Send each character as a key event
	for _, r := range text {
		if err := chromedp.Run(ctx,
			input.DispatchKeyEvent(input.KeyDown).WithText(string(r)),
			input.DispatchKeyEvent(input.KeyUp).WithText(string(r)),
		); err != nil {
			return fmt.Errorf("input failed: %w", err)
		}
	}
	return nil
}

// GetURL returns the current URL.
func (inst *Instance) GetURL() string {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	var loc string
	ctx, cancel := context.WithTimeout(inst.ctx, 5*time.Second)
	defer cancel()

	chromedp.Run(ctx, chromedp.Location(&loc))
	if loc != "" {
		inst.url = loc
	}
	return inst.url
}

// GoBack navigates back in history.
func (inst *Instance) GoBack() error {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := context.WithTimeout(inst.ctx, 15*time.Second)
	defer cancel()

	return chromedp.Run(ctx, chromedp.NavigateBack())
}

// GoForward navigates forward in history.
func (inst *Instance) GoForward() error {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := context.WithTimeout(inst.ctx, 15*time.Second)
	defer cancel()

	return chromedp.Run(ctx, chromedp.NavigateForward())
}

// Eval runs JavaScript and returns the result.
func (inst *Instance) Eval(expression string) (string, error) {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := context.WithTimeout(inst.ctx, 10*time.Second)
	defer cancel()

	var result string
	if err := chromedp.Run(ctx,
		chromedp.EvaluateAsDevTools(expression, &result),
	); err != nil {
		return "", fmt.Errorf("eval failed: %w", err)
	}
	return result, nil
}

// ClickSelector clicks on the element matching the CSS selector.
func (inst *Instance) ClickSelector(selector string) error {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := context.WithTimeout(inst.ctx, 10*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	)
}

// InputSelector fills a form field identified by the CSS selector.
func (inst *Instance) InputSelector(selector, value string) error {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	ctx, cancel := context.WithTimeout(inst.ctx, 10*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.SendKeys(selector, value, chromedp.ByQuery),
	)
}

// _ ensures cdp and chromedp packages are referenced (no unused import errors).
var _ = cdp.Node{}
