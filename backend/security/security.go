// Package security implements tiered YAML permissions + token-gated execution.
package security

// Tier represents a permission level for command execution.
type Tier string

const (
	TierSafe   Tier = "safe"   // Execute immediately
	TierPrompt Tier = "prompt" // Require user approval
	TierBlock  Tier = "block"  // Deny and log
)

// SecurityConfig holds the tiered permission rules.
type SecurityConfig struct {
	Safe   TierConfig `yaml:"safe"`
	Prompt TierConfig `yaml:"prompt"`
	Block  TierConfig `yaml:"block"`
}

// TierConfig defines paths and commands allowed at a given tier.
type TierConfig struct {
	Paths []string `yaml:"paths"`
	Cmds  []string `yaml:"cmds"`
}

// ExecToken is a short-lived JWT granting permission to run a blocked command.
type ExecToken struct {
	Token     string `json:"token"`
	Command   string `json:"command"`
	ExpiresAt int64  `json:"expires_at"`
}
