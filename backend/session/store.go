// Package session provides SQLite-backed session storage.
// Mirrors the session model from hermes-webui (api/models.py) in Go.
package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Message represents an OpenAI-format chat message.
type Message struct {
	Role    string                 `json:"role"` // "user", "assistant", "system", "tool"
	Content string                 `json:"content"`
	ToolCalls []ToolCall           `json:"tool_calls,omitempty"`
	ToolCallID string              `json:"tool_call_id,omitempty"`
	Name    string                 `json:"name,omitempty"` // for tool role
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function FunctionCall           `json:"function"`
}

// FunctionCall is the function portion of a tool call.
type FunctionCall struct {
	Name      string                 `json:"name"`
	Arguments string                 `json:"arguments"` // JSON string
}

// Session is the primary data model — mirrors hermes-webui's Session class.
type Session struct {
	ID             string    `json:"session_id"` // 12-char hex (uuid4[:12])
	Title          string    `json:"title"`       // auto from first message, max 64 chars
	Workspace      string    `json:"workspace"`   // absolute path
	Model          string    `json:"model"`       // e.g. "anthropic/claude-sonnet-4"
	Messages       []Message `json:"messages"`
	CreatedAt      int64     `json:"created_at"`  // Unix timestamp
	UpdatedAt      int64     `json:"updated_at"`  // Unix timestamp
	Pinned         bool      `json:"pinned"`
	Archived       bool      `json:"archived"`
	ProjectID      string    `json:"project_id,omitempty"`
	ToolCalls      []ToolCall `json:"tool_calls,omitempty"`
	InputTokens    int       `json:"input_tokens,omitempty"`    // Phase 6: cost ledger
	OutputTokens   int       `json:"output_tokens,omitempty"`   // Phase 6: cost ledger
	EstimatedCost  float64   `json:"estimated_cost,omitempty"`  // Phase 6: cost ledger in USD
}

// Compact returns a minimal representation for session list views.
func (s *Session) Compact() map[string]interface{} {
	return map[string]interface{}{
		"session_id":    s.ID,
		"title":         s.Title,
		"workspace":     s.Workspace,
		"model":         s.Model,
		"pinned":        s.Pinned,
		"archived":      s.Archived,
		"project_id":    s.ProjectID,
		"created_at":    s.CreatedAt,
		"updated_at":    s.UpdatedAt,
		"message_count": len(s.Messages),
		"input_tokens":  s.InputTokens,
		"output_tokens": s.OutputTokens,
		"estimated_cost": s.EstimatedCost,
	}
}

// Path returns the session's file path.
func (s *Session) Path(dbPath string) string {
	return filepath.Join(dbPath, "sessions", s.ID+".json")
}

// Store is the SQLite-backed session repository.
type Store struct {
	dbPath string
	smu    sync.RWMutex
	sessions map[string]*Session // in-memory cache
	mu      sync.RWMutex
}

// NewStore creates or opens a session store at dbPath.
func NewStore(dbPath string) (*Store, error) {
	s := &Store{dbPath: dbPath, sessions: make(map[string]*Session)}
	if err := os.MkdirAll(filepath.Join(dbPath, "sessions"), 0755); err != nil {
		return nil, fmt.Errorf("session store init: %w", err)
	}
	if err := s.loadIndex(); err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}
	return s, nil
}

// indexPath returns the path to the session index file.
func (s *Store) indexPath() string {
	return filepath.Join(s.dbPath, "sessions", "_index.json")
}

// loadIndex loads session IDs from the index file into memory.
func (s *Store) loadIndex() error {
	idxPath := s.indexPath()
	data, err := os.ReadFile(idxPath)
	if os.IsNotExist(err) {
		return nil // first run, no index yet
	}
	if err != nil {
		return err
	}
	var entries []struct {
		ID        string `json:"session_id"`
		UpdatedAt int64  `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	for _, e := range entries {
		s.mu.Lock()
		s.sessions[e.ID] = &Session{ID: e.ID, UpdatedAt: e.UpdatedAt} // lazy load
		s.mu.Unlock()
	}
	return nil
}

// writeIndex atomically writes the session index.
func (s *Store) writeIndex() error {
	s.mu.RLock()
	type entry struct {
		ID        string `json:"session_id"`
		UpdatedAt int64  `json:"updated_at"`
	}
	var entries []entry
	for id, sess := range s.sessions {
		entries = append(entries, entry{ID: id, UpdatedAt: sess.UpdatedAt})
	}
	s.mu.RUnlock()
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	// Atomic write: write to .tmp then rename
	tmpPath := s.indexPath() + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.indexPath())
}

// New creates a new session with the given workspace and model.
func (s *Store) New(workspace, model string) (*Session, error) {
	// Resolve workspace to absolute path
	if !filepath.IsAbs(workspace) {
		abs, err := filepath.Abs(workspace)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace: %w", err)
		}
		workspace = abs
	}

	sess := &Session{
		ID:        uuid.New().String()[:12],
		Title:     "New conversation",
		Workspace: workspace,
		Model:     model,
		Messages:  []Message{},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Pinned:    false,
		Archived:  false,
	}

	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	if err := s.save(sess); err != nil {
		return nil, fmt.Errorf("save new session: %w", err)
	}
	if err := s.writeIndex(); err != nil {
		return nil, fmt.Errorf("write index: %w", err)
	}

	return sess, nil
}

// Get retrieves a session by ID, loading from disk if not in cache.
func (s *Store) Get(id string) (*Session, error) {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if ok && len(sess.Messages) > 0 {
		return sess, nil
	}

	// Lazy load from disk
	path := filepath.Join(s.dbPath, "sessions", id+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	sess = &Session{}
	if err := json.Unmarshal(data, sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess, nil
}

// GetCompact returns compact dict for a session without loading full messages.
func (s *Store) GetCompact(id string) (map[string]interface{}, error) {
	sess, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	return sess.Compact(), nil
}

// save writes the session to disk atomically.
func (s *Store) save(sess *Session) error {
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	path := sess.Path(s.dbPath)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return os.Rename(tmpPath, path)
}

// Save persists a session to disk and updates the index.
func (s *Store) Save(sess *Session) error {
	s.mu.Lock()
	sess.UpdatedAt = time.Now().Unix()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	if err := s.save(sess); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	return s.writeIndex()
}

// Delete removes a session from disk and memory.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()

	path := filepath.Join(s.dbPath, "sessions", id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete: %w", err)
	}
	return s.writeIndex()
}

// All returns all sessions sorted by UpdatedAt descending.
func (s *Store) All() ([]*Session, error) {
	s.mu.RLock()
	var result []*Session
	for _, sess := range s.sessions {
		if sess.Title == "" && len(sess.Messages) == 0 {
		}
		result = append(result, sess)
	}
	s.mu.RUnlock()
	// Sort by UpdatedAt descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].UpdatedAt > result[i].UpdatedAt {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

// AllCompact returns compact representations of all sessions.
func (s *Store) AllCompact() ([]map[string]interface{}, error) {
	sessions, err := s.All()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(sessions))
	for _, sess := range sessions {
		result = append(result, sess.Compact())
	}
	return result, nil
}

// Search performs full-text search on session titles and messages.
func (s *Store) Search(query string) ([]map[string]interface{}, error) {
	sessions, err := s.All()
	if err != nil {
		return nil, err
	}
	queryLower := toLower(query)
	var results []map[string]interface{}
	for _, sess := range sessions {
		if containsLower(sess.Title, queryLower) {
			results = append(results, sess.Compact())
			continue
		}
		// Search message content
		for _, msg := range sess.Messages {
			if containsLower(msg.Content, queryLower) {
				results = append(results, sess.Compact())
				break
			}
		}
	}
	return results, nil
}

// UpdateTitle sets the session title from the first user message.
func (s *Store) UpdateTitle(id string) error {
	sess, err := s.Get(id)
	if err != nil {
		return err
	}
	for _, msg := range sess.Messages {
		if msg.Role == "user" {
			title := msg.Content
			if len(title) > 64 {
				title = title[:64]
			}
			sess.Title = title
			break
		}
	}
	return s.Save(sess)
}

// Pin sets the pinned state of a session.
func (s *Store) Pin(id string, pinned bool) error {
	sess, err := s.Get(id)
	if err != nil {
		return err
	}
	sess.Pinned = pinned
	return s.Save(sess)
}

// Archive sets the archived state of a session.
func (s *Store) Archive(id string, archived bool) error {
	sess, err := s.Get(id)
	if err != nil {
		return err
	}
	sess.Archived = archived
	return s.Save(sess)
}

// AddMessage appends a message to a session.
func (s *Store) AddMessage(id string, msg Message) error {
	sess, err := s.Get(id)
	if err != nil {
		return err
	}
	sess.Messages = append(sess.Messages, msg)
	return s.Save(sess)
}

// SetMessages replaces a session's message array.
func (s *Store) SetMessages(id string, msgs []Message) error {
	sess, err := s.Get(id)
	if err != nil {
		return err
	}
	sess.Messages = msgs
	return s.Save(sess)
}

// DeleteMessagesAfter removes all messages from index onward.
func (s *Store) DeleteMessagesAfter(id string, afterIndex int) error {
	sess, err := s.Get(id)
	if err != nil {
		return err
	}
	if afterIndex < 0 || afterIndex >= len(sess.Messages) {
		sess.Messages = []Message{}
	} else {
		sess.Messages = sess.Messages[:afterIndex+1]
	}
	return s.Save(sess)
}

// Hash returns a stable hash of the session messages for diff-aware resume.
func (s *Store) Hash(id string) (string, error) {
	sess, err := s.Get(id)
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(sess.Messages)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// Helpers

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func containsLower(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}