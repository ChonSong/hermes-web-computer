// Package security implements tiered YAML permissions + token-gated execution.
// Commands are classified into safe/prompt/block tiers based on path and
// command matching. Time-limited tokens grant one-time execution approval.
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Config holds the tiered permission rules loaded from YAML.
type Config struct {
	Tiers map[string]Tier `yaml:"tiers"`
}

// Tier defines paths and commands allowed at a given permission level.
type Tier struct {
	Paths []string `yaml:"paths"`
	Cmds  []string `yaml:"cmds"`
}

// Enforcer evaluates commands against tier rules and manages tokens.
type Enforcer struct {
	mu     sync.RWMutex
	config Config
	tokens map[string]*Token // token -> {cmd, expiresAt}
}

// Token is a short-lived grant permitting execution of a specific command.
type Token struct {
	Command   string
	ExpiresAt time.Time
}

// NewEnforcer creates an enforcer with default safe configuration.
func NewEnforcer() *Enforcer {
	e := &Enforcer{
		tokens: make(map[string]*Token),
	}
	e.config = e.DefaultConfig()
	return e
}

// UseDefaults resets the enforcer to its default safe configuration.
func (e *Enforcer) UseDefaults() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = e.DefaultConfig()
}

// ValidateAndConsume checks if a token is valid (ignoring command match for approval flow).
func (e *Enforcer) ValidateAndConsume(token string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, ok := e.tokens[token]
	if !ok {
		return false
	}

	// Check expiry
	if time.Now().After(t.ExpiresAt) {
		delete(e.tokens, token)
		return false
	}

	// Token is valid — consume it (single-use)
	delete(e.tokens, token)
	return true
}

// GetTokenCommand retrieves the command associated with a token.
func (e *Enforcer) GetTokenCommand(token string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	t, ok := e.tokens[token]
	if !ok {
		return ""
	}
	return t.Command
}

// LoadConfig reads and parses a YAML configuration file.
func (e *Enforcer) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("security: read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("security: parse config: %w", err)
	}

	e.mu.Lock()
	e.config = cfg
	e.mu.Unlock()
	return nil
}

// DefaultConfig returns a safe default configuration.
func (e *Enforcer) DefaultConfig() Config {
	return Config{
		Tiers: map[string]Tier{
			"safe": {
				Paths: []string{"/agent/workspace/**"},
				Cmds:  []string{"ls", "cat", "git status", "pip install", "npm install", "go build", "make"},
			},
			"prompt": {
				Paths: []string{"/host/project/**"},
				Cmds:  []string{"git commit", "npm run build", "docker build"},
			},
			"block": {
				Cmds: []string{
					"rm -rf", "chmod 777", "curl | bash", "sudo", "dd",
					"mkfs", "mkfs.*", "shutdown", "reboot", "kill -9",
				},
			},
		},
	}
}

// Classify evaluates a command and working directory against tier rules.
// Returns the tier names and an error if the command is blocked.
func (e *Enforcer) Classify(cmd string, cwd string) (tier string, err error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check block tier first (highest priority)
	if blockTier, ok := e.config.Tiers["block"]; ok {
		if matchCommand(cmd, blockTier.Cmds) || matchPath(cwd, blockTier.Paths) {
			return "block", fmt.Errorf("security: command %q is blocked", cmd)
		}
	}

	// Check safe tier
	if safeTier, ok := e.config.Tiers["safe"]; ok {
		if matchCommand(cmd, safeTier.Cmds) || matchPath(cwd, safeTier.Paths) {
			return "safe", nil
		}
	}

	// Check prompt tier
	if promptTier, ok := e.config.Tiers["prompt"]; ok {
		if matchCommand(cmd, promptTier.Cmds) || matchPath(cwd, promptTier.Paths) {
			return "prompt", nil
		}
	}

	// Default: safe for dev environments (unknown commands allowed)
	return "safe", nil
}

// GrantToken generates a one-time execution token for a command.
// Returns the token string and its expiry time (30s from now).
func (e *Enforcer) GrantToken(cmd string) (string, time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()

	token := uuid.New().String()
	expiresAt := time.Now().Add(30 * time.Second)

	e.tokens[token] = &Token{
		Command:   cmd,
		ExpiresAt: expiresAt,
	}

	// Clean up expired tokens periodically
	e.cleanupExpired()

	return token, expiresAt
}

// ValidateToken checks if a token is valid for the given command.
// Tokens are single-use and expire after 30 seconds.
func (e *Enforcer) ValidateToken(token string, cmd string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, ok := e.tokens[token]
	if !ok {
		return false
	}

	// Check expiry
	if time.Now().After(t.ExpiresAt) {
		delete(e.tokens, token)
		return false
	}

	// Check command match
	if t.Command != cmd {
		return false
	}

	// Token is valid — consume it (single-use)
	delete(e.tokens, token)
	return true
}

// cleanupExpired removes expired tokens. Must be called under lock.
func (e *Enforcer) cleanupExpired() {
	now := time.Now()
	for tok, t := range e.tokens {
		if now.After(t.ExpiresAt) {
			delete(e.tokens, tok)
		}
	}
}

// matchCommand checks if the command matches any pattern in the list.
// Uses exact match for simple commands, path.Match for glob patterns.
func matchCommand(cmd string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == cmd {
			return true
		}
		// Try glob match for patterns like "mkfs.*"
		if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
			if matched, _ := filepath.Match(pattern, cmd); matched {
				return true
			}
		}
	}
	return false
}

// matchPath checks if the path matches any glob pattern in the list.
func matchPath(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if path == "" {
			continue
		}
		// Convert ** patterns for matching
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// Handle ** glob by checking prefix
		if strings.Contains(pattern, "**") {
			prefix := strings.TrimSuffix(pattern, "**")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}
