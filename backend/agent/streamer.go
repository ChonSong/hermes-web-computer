package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StreamEvent represents a token or signal from the agent stream.
type StreamEvent struct {
	Type     string      `json:"type"`
	Content  string      `json:"content,omitempty"`
	ID       string      `json:"id,omitempty"`
	ToolCall *ToolCall   `json:"tool_call,omitempty"`
	Result   string      `json:"result,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// ToolCall represents a function call from the model.
type ToolCall struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Args     map[string]interface{} `json:"arguments"`
}

// Streamer calls the Hermes Agent SSE endpoint and yields events.
type Streamer struct {
	baseURL       string
	client        *http.Client
	sessionCookie string
}

// NewStreamer creates a new agent streamer.
func NewStreamer(baseURL, sessionCookie string) *Streamer {
	if baseURL == "" {
		baseURL = "http://localhost:8787"
	}
	return &Streamer{
		baseURL:       baseURL,
		sessionCookie: sessionCookie,
		client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{DisableKeepAlives: true},
		},
	}
}

// Stream starts an agent conversation and sends events to the callback.
// It reads SSE from hermes-agent and parses token/reasoning/tool events.
func (s *Streamer) Stream(ctx context.Context, message string, onEvent func(StreamEvent)) error {
	reqBody := map[string]interface{}{
		"session_id": "hwc-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		"message":    message,
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/chat/start",
		strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.sessionCookie != "" {
		req.Header.Set("Cookie", s.sessionCookie)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("agent request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("agent returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read SSE stream line by line
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var partial strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(strings.TrimSpace(line), "event: ")
			if scanner.Scan() {
				dataLine := scanner.Text()
				if strings.HasPrefix(dataLine, "data: ") {
					dataStr := strings.TrimPrefix(dataLine, "data: ")
					s.parseSSELine(eventType, dataStr, &partial, onEvent)
				}
			}
		} else if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")
			s.parseSSELine("", dataStr, &partial, onEvent)
		}
	}

	if partial.Len() > 0 {
		onEvent(StreamEvent{Type: "token", Content: partial.String()})
	}
	onEvent(StreamEvent{Type: "stream_end"})
	return nil
}

// parseSSELine dispatches a parsed SSE data payload to onEvent.
func (s *Streamer) parseSSELine(eventType, dataStr string, partial *strings.Builder, onEvent func(StreamEvent)) {
	if dataStr == "" || dataStr == "[DONE]" {
		return
	}

	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &msg); err != nil {
		partial.WriteString(dataStr)
		return
	}

	switch eventType {
	case "token":
		if text, ok := msg["text"].(string); ok {
			if partial.Len() > 0 {
				onEvent(StreamEvent{Type: "token", Content: partial.String()})
				partial.Reset()
			}
			onEvent(StreamEvent{Type: "token", Content: text})
		}
	case "reasoning":
		if text, ok := msg["text"].(string); ok {
			onEvent(StreamEvent{Type: "reasoning", Content: text})
		}
	case "interim_assistant":
		if text, ok := msg["text"].(string); ok {
			onEvent(StreamEvent{Type: "interim", Content: text})
		}
	case "tool_call":
		if tc, ok := msg["tool_calls"].([]interface{}); ok && len(tc) > 0 {
			for _, tcv := range tc {
				if tcm, ok := tcv.(map[string]interface{}); ok {
					fn := tcm["function"].(map[string]interface{})
					onEvent(StreamEvent{
						Type: "tool_call",
						ToolCall: &ToolCall{
							ID:   tcm["id"].(string),
							Name: fn["name"].(string),
							Args: fn["arguments"].(map[string]interface{}),
						},
					})
				}
			}
		}
	case "tool_result":
		if result, ok := msg["result"].(string); ok {
			onEvent(StreamEvent{Type: "tool_result", Result: result})
		}
	case "stream_end", "error", "cancel":
		onEvent(StreamEvent{Type: "stream_end"})
	default:
		if text, ok := msg["text"].(string); ok && text != "" {
			if reason, ok := msg["reasoning"].(string); ok {
				onEvent(StreamEvent{Type: "reasoning", Content: reason})
			} else {
				if partial.Len() > 0 {
					onEvent(StreamEvent{Type: "token", Content: partial.String()})
					partial.Reset()
				}
				onEvent(StreamEvent{Type: "token", Content: text})
			}
		}
	}
}
