// Package telemetry implements a local-first JSONL ring buffer with async cloud sync.
// Events are appended to a local file, automatically pruned when exceeding size limits,
// and periodically synced to a remote endpoint with exponential backoff.
package telemetry

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Event is a single telemetry event.
type Event struct {
	Ts         int64   `json:"ts"`
	SessionID  string  `json:"session"`
	Type       string  `json:"event"`
	User       string  `json:"user,omitempty"`
	Policy     string  `json:"policy,omitempty"`
	DriftScore float64 `json:"drift_score,omitempty"`
	Command    string  `json:"cmd,omitempty"`
	Token      string  `json:"token,omitempty"`
	Outcome    string  `json:"outcome,omitempty"`
	Tool       string  `json:"tool,omitempty"`
	Path       string  `json:"path,omitempty"`
	Size       int     `json:"size,omitempty"`
}

// RingBuffer is a JSONL telemetry buffer with automatic pruning.
type RingBuffer struct {
	path  string
	maxMB int
	mu    sync.Mutex
	file  *os.File
	count int
}

// Syncer manages background sync of telemetry events to a remote endpoint.
type Syncer struct {
	buffer   *RingBuffer
	endpoint string
	interval time.Duration
	backoff  time.Duration
	done     chan struct{}
}

// NewRingBuffer creates a telemetry ring buffer at the given path.
func NewRingBuffer(path string, maxMB int) (*RingBuffer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("telemetry: open buffer file: %w", err)
	}
	return &RingBuffer{
		path:  path,
		maxMB: maxMB,
		file:  f,
	}, nil
}

// Write appends an event to the JSONL buffer.
func (rb *RingBuffer) Write(event Event) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	event.Ts = time.Now().UnixMilli()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("telemetry: marshal event: %w", err)
	}

	if _, err := rb.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("telemetry: write event: %w", err)
	}

	rb.count++

	// Prune if file exceeds max size
	return rb.pruneLocked()
}

// Prune removes oldest lines when the file exceeds maxMB.
func (rb *RingBuffer) Prune() error {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.pruneLocked()
}

// pruneLocked must be called under rb.mu.
func (rb *RingBuffer) pruneLocked() error {
	if rb.maxMB <= 0 {
		return nil
	}

	info, err := rb.file.Stat()
	if err != nil {
		return fmt.Errorf("telemetry: stat buffer: %w", err)
	}

	maxBytes := int64(rb.maxMB) * 1024 * 1024
	if info.Size() <= maxBytes {
		return nil
	}

	// Close current file
	if err := rb.file.Close(); err != nil {
		return fmt.Errorf("telemetry: close before prune: %w", err)
	}

	// Read all lines
	lines, err := readLines(rb.path)
	if err != nil {
		// Reopen file even if prune fails
		rb.file, _ = os.OpenFile(rb.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		return fmt.Errorf("telemetry: read lines for prune: %w", err)
	}

	// Keep only the most recent lines that fit within maxBytes
	kept := trimToSize(lines, maxBytes)

	// Rewrite file with kept lines
	if err := os.WriteFile(rb.path, nil, 0644); err != nil {
		rb.file, _ = os.OpenFile(rb.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		return fmt.Errorf("telemetry: truncate for prune: %w", err)
	}

	if err := os.WriteFile(rb.path, []byte(""), 0644); err != nil {
		rb.file, _ = os.OpenFile(rb.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		return fmt.Errorf("telemetry: rewrite buffer: %w", err)
	}

	f, err := os.OpenFile(rb.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("telemetry: reopen after prune: %w", err)
	}

	for _, line := range kept {
		if _, err := f.Write(append(line, '\n')); err != nil {
			f.Close()
			return fmt.Errorf("telemetry: write pruned line: %w", err)
		}
	}

	rb.file = f
	rb.count = len(kept)
	return nil
}

// readLines reads all lines from a file.
func readLines(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	// Increase buffer size for potentially long JSON lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := make([]byte, len(scanner.Bytes()))
		copy(line, scanner.Bytes())
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

// trimToSize returns the tail of lines that fit within maxBytes.
func trimToSize(lines [][]byte, maxBytes int64) [][]byte {
	var total int64
	for _, line := range lines {
		total += int64(len(line)) + 1 // +1 for newline
	}

	if total <= maxBytes {
		return lines
	}

	// Remove lines from the front until under limit
	start := 0
	for start < len(lines) && total > maxBytes {
		total -= int64(len(lines[start])) + 1
		start++
	}

	return lines[start:]
}

// ReadLast returns the last n events from the buffer.
func (rb *RingBuffer) ReadLast(n int) ([]Event, error) {
	rb.mu.Lock()

	// Flush and close current file for reading
	if rb.file != nil {
		rb.file.Sync()
	}
	rb.mu.Unlock()

	f, err := os.Open(rb.path)
	if err != nil {
		return nil, fmt.Errorf("telemetry: open for read: %w", err)
	}
	defer f.Close()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := make([]byte, len(scanner.Bytes()))
		copy(line, scanner.Bytes())
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("telemetry: scan lines: %w", err)
	}

	// Take last n lines
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	lines = lines[start:]

	events := make([]Event, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var evt Event
		if err := json.Unmarshal(line, &evt); err != nil {
			continue // Skip malformed lines
		}
		events = append(events, evt)
	}

	return events, nil
}

// NewSyncer creates a syncer that periodically uploads events to the endpoint.
func NewSyncer(buffer *RingBuffer, endpoint string) *Syncer {
	return &Syncer{
		buffer:   buffer,
		endpoint: endpoint,
		interval: 30 * time.Second,
		backoff:  1 * time.Second,
		done:     make(chan struct{}),
	}
}

// Start begins the background sync goroutine.
func (s *Syncer) Start() {
	go s.run()
}

// run is the main sync loop with exponential backoff.
func (s *Syncer) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			if err := s.syncOnce(); err != nil {
				// Exponential backoff: 1s, 2s, 4s, 8s, max 60s
				s.backoff *= 2
				if s.backoff > 60*time.Second {
					s.backoff = 60 * time.Second
				}
			} else {
				// Reset backoff on success
				s.backoff = 1 * time.Second
				// Prune buffer after successful sync
				_ = s.buffer.Prune()
			}

			// Wait for backoff before next attempt (unless done)
			select {
			case <-s.done:
				return
			case <-time.After(s.backoff):
			}
		}
	}
}

// syncOne reads all events and POSTs them to the endpoint.
func (s *Syncer) syncOnce() error {
	events, err := s.buffer.ReadLast(1000)
	if err != nil {
		return fmt.Errorf("telemetry: read events: %w", err)
	}

	if len(events) == 0 {
		return nil // Nothing to sync
	}

	data, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("telemetry: marshal events: %w", err)
	}

	resp, err := http.Post(s.endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("telemetry: POST to %s: %w", s.endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telemetry: sync failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Stop signals the syncer to shut down.
func (s *Syncer) Stop() {
	close(s.done)
}
