package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the Hermes config.yaml structure
type Config struct {
	Model       ModelConfig `json:"model" yaml:"model"`
	Providers   interface{} `json:"providers" yaml:"providers"`
	Fallback    []string     `json:"fallback_providers" yaml:"fallback_providers"`
	Toolsets    []string     `json:"toolsets" yaml:"toolsets"`
	Agent       AgentConfig `json:"agent" yaml:"agent"`
	Terminal    interface{} `json:"terminal" yaml:"terminal"`
	Web         interface{} `json:"web" yaml:"web"`
	Browser     interface{} `json:"browser" yaml:"browser"`
	Environment EnvConfig   `json:"env" yaml:"env"`
}

// ModelConfig holds model settings
type ModelConfig struct {
	BaseURL  string `json:"base_url" yaml:"base_url"`
	Default  string `json:"default" yaml:"default"`
	Provider string `json:"provider" yaml:"provider"`
	APIKey   string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
}

// AgentConfig holds agent settings
type AgentConfig struct {
	MaxTurns              int               `json:"max_turns" yaml:"max_turns"`
	GatewayTimeout        int               `json:"gateway_timeout" yaml:"gateway_timeout"`
	RestartDrainTimeout   int               `json:"restart_drain_timeout" yaml:"restart_drain_timeout"`
	APIMaxRetries         int               `json:"api_max_retries" yaml:"api_max_retries"`
	ServiceTier           string            `json:"service_tier" yaml:"service_tier"`
	ToolUseEnforcement    string            `json:"tool_use_enforcement" yaml:"tool_use_enforcement"`
	GatewayTimeoutWarning int               `json:"gateway_timeout_warning" yaml:"gateway_timeout_warning"`
	GatewayNotifyInterval int               `json:"gateway_notify_interval" yaml:"gateway_notify_interval"`
	GatewayAutoContinue   int               `json:"gateway_auto_continue_freshness" yaml:"gateway_auto_continue_freshness"`
	ImageInputMode        string            `json:"image_input_mode" yaml:"image_input_mode"`
	DisabledToolsets      []string          `json:"disabled_toolsets" yaml:"disabled_toolsets"`
	Personalities         map[string]string `json:"personalities" yaml:"personalities"`
	ReasoningEffort       string            `json:"reasoning_effort" yaml:"reasoning_effort"`
	Verbose               bool              `json:"verbose" yaml:"verbose"`
}

// EnvConfig holds environment variables from config.yaml
type EnvConfig struct {
	Vars map[string]string `json:"vars" yaml:"vars"`
}

// Manager handles reading/writing config.yaml and env vars
type Manager struct {
	mu       sync.RWMutex
	config   *Config
	configPath string
}

// NewManager creates a new config manager
func NewManager() (*Manager, error) {
	m := &Manager{
		configPath: os.Getenv("HERMES_CONFIG"),
	}
	if m.configPath == "" {
		home, _ := os.UserHomeDir()
		m.configPath = filepath.Join(home, ".hermes", "config.yaml")
	}
	if err := m.Load(); err != nil {
		return nil, err
	}
	return m, nil
}

// Load reads config.yaml from disk
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Initialize env vars map if nil
	if cfg.Environment.Vars == nil {
		cfg.Environment.Vars = make(map[string]string)
	}

	m.config = &cfg
	return nil
}

// Save writes config.yaml to disk
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Get returns the current config
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Set updates a config value by key path (e.g. "model.default")
func (m *Manager) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch key {
	case "model.base_url":
		m.config.Model.BaseURL = value.(string)
	case "model.default":
		m.config.Model.Default = value.(string)
	case "model.provider":
		m.config.Model.Provider = value.(string)
	case "model.api_key":
		m.config.Model.APIKey = value.(string)
	case "agent.max_turns":
		m.config.Agent.MaxTurns = int(value.(float64))
	case "agent.reasoning_effort":
		m.config.Agent.ReasoningEffort = value.(string)
	case "agent.verbose":
		m.config.Agent.Verbose = value.(bool)
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}

// Delete removes a config value by key path
func (m *Manager) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch key {
	case "model.api_key":
		m.config.Model.APIKey = ""
	default:
		return fmt.Errorf("cannot delete key: %s", key)
	}

	return nil
}

// EnvList returns all environment variables
func (m *Manager) EnvList() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Environment.Vars
}

// EnvSet sets an environment variable
func (m *Manager) EnvSet(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.config.Environment.Vars == nil {
		m.config.Environment.Vars = make(map[string]string)
	}
	m.config.Environment.Vars[key] = value
	return nil
}

// EnvDelete removes an environment variable
func (m *Manager) EnvDelete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.config.Environment.Vars != nil {
		delete(m.config.Environment.Vars, key)
	}
	return nil
}

// RestartSignal signals the agent to restart
func (m *Manager) RestartSignal() error {
	// Touch a restart flag file
	home, _ := os.UserHomeDir()
	flagPath := filepath.Join(home, ".hermes", "restart.flag")
	return os.WriteFile(flagPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
}

// ToJSON returns config as JSON
func (m *Manager) ToJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return json.Marshal(m.config)
}

// Global manager instance
var (
	globalMgr     *Manager
	globalMgrOnce sync.Once
)

// GetManager returns the singleton config manager
func GetManager() (*Manager, error) {
	var initErr error
	globalMgrOnce.Do(func() {
		globalMgr, initErr = NewManager()
	})
	return globalMgr, initErr
}