// Package mcp provides a client for connecting to MCP servers via stdio.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Protocol constants
const (
	ProtocolVersion = "2024-11-05"
)

// JSONRPCMessage represents a JSON-RPC 2.0 message.
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Client represents an MCP client connection to a server.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser
	
	mu       sync.RWMutex
	pending  map[interface{}]chan *JSONRPCMessage
	reqID    int64
	running  bool
	
	handlers map[string]func(*JSONRPCMessage)
}

// ServerInfo contains information about an MCP server.
type ServerInfo struct {
	Name    string                 `json:"name"`
	Version string                 `json:"version"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
}

// Tool represents an MCP tool.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Resource represents an MCP resource.
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
}

// ResourceTemplate represents an MCP resource template.
type ResourceTemplate struct {
	URI         string                 `json:"uriTemplate"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
}

// Prompt represents an MCP prompt.
type Prompt struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt argument.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// InitializeResult is the result of an initialize request.
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities     ServerCapabilities `json:"capabilities"`
	ServerInfo       ServerInfo   `json:"serverInfo"`
}

// ServerCapabilities represents server capabilities.
type ServerCapabilities struct {
	Tools         *struct{} `json:"tools,omitempty"`
	Resources     *struct{} `json:"resources,omitempty"`
	Prompts       *struct{} `json:"prompts,omitempty"`
}

// NewClient creates a new MCP client connecting to a server at the given command.
func NewClient(command string, args ...string) *Client {
	return &Client{
		cmd:      exec.Command(command, args...),
		pending:  make(map[interface{}]chan *JSONRPCMessage),
		handlers: make(map[string]func(*JSONRPCMessage)),
		reqID:    1,
	}
}

// Start launches the MCP server process.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("client already running")
	}
	
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stdin = stdin
	
	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	c.stdout = bufio.NewReader(stdout)
	
	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	c.stderr = stderr

	if err := c.cmd.Start(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to start server: %w", err)
	}
	
	c.running = true
	c.mu.Unlock()

	// Read stderr in background (for debugging/logging)
	go func() {
		scanner := bufio.NewScanner(c.stderr)
		for scanner.Scan() {
			// Could log this if configured
			_ = scanner.Text()
		}
	}()

	return nil
}

// Stop terminates the MCP server process.
func (c *Client) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.running {
		return nil
	}
	
	c.running = false
	
	// Close stdin to signal EOF
	if c.stdin != nil {
		c.stdin.Close()
	}
	
	// Wait for process to exit
	c.cmd.Wait()
	
	// Clean up pending requests
	for _, ch := range c.pending {
		close(ch)
	}
	c.pending = make(map[interface{}]chan *JSONRPCMessage)
	
	return nil
}

// Send sends a JSON-RPC request and waits for a response.
func (c *Client) Send(ctx context.Context, method string, params interface{}) (*JSONRPCMessage, error) {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil, fmt.Errorf("client not running")
	}
	
	reqID := c.reqID
	c.reqID++
	
	respCh := make(chan *JSONRPCMessage, 1)
	c.pending[reqID] = respCh
	c.mu.Unlock()
	
	defer func() {
		c.mu.Lock()
		delete(c.pending, reqID)
		c.mu.Unlock()
	}()

	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		ID:      reqID,
	}
	
	if params != nil {
		payload, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		msg.Params = payload
	}
	
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Add newline for JSON-RPC framing
	payload = append(payload, '\n')
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		if _, err := c.stdin.Write(payload); err != nil {
			return nil, fmt.Errorf("failed to write message: %w", err)
		}
	}
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("JSON-RPC error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
		}
		return resp, nil
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// Initialize initializes the MCP server connection.
func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
	params := map[string]interface{}{
		"protocolVersion": ProtocolVersion,
		"clientInfo": map[string]interface{}{
			"name":    "hermes-web-computer",
			"version": "1.0.0",
		},
	}
	
	resp, err := c.Send(ctx, "initialize", params)
	if err != nil {
		return nil, err
	}
	
	var result InitializeResult
	if resp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}
	
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	
	// Send initialized notification
	c.SendNotification(ctx, "initialized", map[string]interface{}{})
	
	return &result, nil
}

// SendNotification sends a JSON-RPC notification (no response expected).
func (c *Client) SendNotification(ctx context.Context, method string, params interface{}) error {
	c.mu.RLock()
	if !c.running {
		c.mu.RUnlock()
		return fmt.Errorf("client not running")
	}
	c.mu.RUnlock()

	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
	}
	
	if params != nil {
		payload, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
		msg.Params = payload
	}
	
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	payload = append(payload, '\n')
	
	_, err = c.stdin.Write(payload)
	return err
}

// ListTools returns the list of available tools from the server.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	resp, err := c.Send(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}
	
	if resp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}
	
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, err
	}
	
	var listResp struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(resultData, &listResp); err != nil {
		return nil, err
	}
	
	return listResp.Tools, nil
}

// CallTool calls a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	}
	
	resp, err := c.Send(ctx, "tools/call", params)
	if err != nil {
		return nil, err
	}
	
	if resp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}
	
	return resp.Result, nil
}

// ListResources returns the list of available resources from the server.
func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	resp, err := c.Send(ctx, "resources/list", nil)
	if err != nil {
		return nil, err
	}
	
	if resp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}
	
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, err
	}
	
	var listResp struct {
		Resources []Resource `json:"resources"`
	}
	if err := json.Unmarshal(resultData, &listResp); err != nil {
		return nil, err
	}
	
	return listResp.Resources, nil
}

// ListResourceTemplates returns the list of resource templates.
func (c *Client) ListResourceTemplates(ctx context.Context) ([]ResourceTemplate, error) {
	resp, err := c.Send(ctx, "resources/templates/list", nil)
	if err != nil {
		return nil, err
	}
	
	if resp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}
	
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, err
	}
	
	var listResp struct {
		Templates []ResourceTemplate `json:"resourceTemplates"`
	}
	if err := json.Unmarshal(resultData, &listResp); err != nil {
		return nil, err
	}
	
	return listResp.Templates, nil
}

// ReadResource reads a resource from the server.
func (c *Client) ReadResource(ctx context.Context, uri string) (interface{}, error) {
	params := map[string]interface{}{
		"uri": uri,
	}
	
	resp, err := c.Send(ctx, "resources/read", params)
	if err != nil {
		return nil, err
	}
	
	if resp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}
	
	return resp.Result, nil
}

// ListPrompts returns the list of available prompts from the server.
func (c *Client) ListPrompts(ctx context.Context) ([]Prompt, error) {
	resp, err := c.Send(ctx, "prompts/list", nil)
	if err != nil {
		return nil, err
	}
	
	if resp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}
	
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, err
	}
	
	var listResp struct {
		Prompts []Prompt `json:"prompts"`
	}
	if err := json.Unmarshal(resultData, &listResp); err != nil {
		return nil, err
	}
	
	return listResp.Prompts, nil
}

// GetPrompt gets a rendered prompt from the server.
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	}
	
	resp, err := c.Send(ctx, "prompts/get", params)
	if err != nil {
		return nil, err
	}
	
	if resp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}
	
	return resp.Result, nil
}

// SubscribeToNotifications starts listening for server notifications.
func (c *Client) SubscribeToNotifications(handler func(method string, params interface{})) {
	c.mu.Lock()
	c.handlers["notification"] = func(msg *JSONRPCMessage) {
		if msg.Method != "" && msg.ID == nil {
			var params interface{}
			if msg.Params != nil {
				json.Unmarshal(msg.Params, &params)
			}
			handler(msg.Method, params)
		}
	}
	c.mu.Unlock()
	
	// Start reading notifications in background
	go c.readNotifications()
}

// readNotifications continuously reads messages from stdout and dispatches them.
func (c *Client) readNotifications() {
	for {
		c.mu.RLock()
		if !c.running {
			c.mu.RUnlock()
			return
		}
		handler := c.handlers["notification"]
		c.mu.RUnlock()
		
		if handler == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		
		var msg JSONRPCMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		
		handler(&msg)
	}
}

// SetRequestHandler sets a handler for incoming requests from the server.
func (c *Client) SetRequestHandler(method string, handler func(params interface{}) (interface{}, error)) {
	c.mu.Lock()
	c.handlers[method] = func(msg *JSONRPCMessage) {
		var params interface{}
		if msg.Params != nil {
			json.Unmarshal(msg.Params, &params)
		}
		
		result, err := handler(params)
		if err != nil {
			c.sendErrorResponse(msg.ID, -32603, err.Error())
			return
		}
		
		c.sendResponse(msg.ID, result)
	}
	c.mu.Unlock()
}

// sendResponse sends a JSON-RPC response.
func (c *Client) sendResponse(id interface{}, result interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	resp := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	
	_, err = c.stdin.Write(payload)
	return err
}

// sendErrorResponse sends a JSON-RPC error response.
func (c *Client) sendErrorResponse(id interface{}, code int, message string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	resp := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
	
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	
	_, err = c.stdin.Write(payload)
	return err
}

// ReadLoop starts the main read loop to process incoming messages.
func (c *Client) ReadLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}
		
		var msg JSONRPCMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		
		// Handle response to our request
		if msg.ID != nil {
			c.mu.RLock()
			ch, ok := c.pending[msg.ID]
			c.mu.RUnlock()
			
			if ok {
				select {
				case ch <- &msg:
				default:
				}
				continue
			}
		}
		
		// Handle notification or server-initiated request
		if msg.Method != "" {
			c.mu.RLock()
			handler, ok := c.handlers[msg.Method]
			c.mu.RUnlock()
			
			if ok {
				handler(&msg)
			}
		}
	}
}

// Manager manages multiple MCP client connections.
type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewManager creates a new MCP manager.
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// AddClient adds a new MCP client with the given name.
func (m *Manager) AddClient(name string, command string, args ...string) *Client {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	client := NewClient(command, args...)
	m.clients[name] = client
	return client
}

// GetClient returns a client by name.
func (m *Manager) GetClient(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	client, ok := m.clients[name]
	return client, ok
}

// RemoveClient stops and removes a client.
func (m *Manager) RemoveClient(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	client, ok := m.clients[name]
	if !ok {
		return fmt.Errorf("client not found: %s", name)
	}
	
	client.Stop()
	delete(m.clients, name)
	return nil
}

// ListClients returns all client names.
func (m *Manager) ListClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// Close stops all clients.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, client := range m.clients {
		client.Stop()
	}
	m.clients = make(map[string]*Client)
	return nil
}