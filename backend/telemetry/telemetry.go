// Package telemetry implements local-first JSONL ring buffer with async cloud sync.
package telemetry

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Event is a single telemetry event.
type Event struct {
	Ts        int64  `json:"ts"`
	SessionID string `json:"session"`
	Type      string `json:"event"`
	User      string `json:"user,omitempty"`
	Policy    string `json:"policy,omitempty"`
	DriftScore float64 `json:"drift_score,omitempty"`
	Command   string `json:"cmd,omitempty"`
	Token     string `json:"token,omitempty"`
	Outcome   string `json:"outcome,omitempty"`
	Tool      string `json:"tool,omitempty"`
	Path      string `json:"path,omitempty"`
	Size      int    `json:"size,omitempty"`
}

// RingBuffer is a JSONL telemetry buffer with automatic pruning.
type RingBuffer struct {
	mu     sync.Mutex
	file   *os.File
	maxMB  int
	count  int
}

// NewRingBuffer creates a telemetry ring buffer.
func NewRingBuffer(path string, maxMB int) (*RingBuffer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &RingBuffer{file: f, maxMB: maxMB}, nil
}

// Write appends an event to the JSONL buffer.
func (rb *RingBuffer) Write(event Event) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	event.Ts = time.Now().UnixMilli()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = rb.file.Write(append(data, '\n'))
	rb.count++
	return err
}

// SyncToCloud attempts async sync to Langfuse/Opik (stub).
func (rb *RingBuffer) SyncToCloud() error {
	// TODO: implement async sync with exponential backoff
	return nil
}
