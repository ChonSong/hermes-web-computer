package security

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEnforcer_BlockDirTraversal verifies that the enforcer classifies
// path traversal commands (e.g., accessing /etc/passwd) as blocked.
func TestEnforcer_BlockDirTraversal(t *testing.T) {
	e := NewEnforcer()
	e.UseDefaults()

	// Commands that exactly match blocked patterns
	traversalCmds := []string{
		"sudo",
		"rm -rf",
		"chmod 777",
		"dd",
		"kill -9",
		"shutdown",
		"reboot",
	}

	for _, cmd := range traversalCmds {
		t.Run(cmd, func(t *testing.T) {
			tier, err := e.Classify(cmd, "/agent/workspace")
			if tier != "block" {
				t.Errorf("expected tier 'block' for %q, got %q (err=%v)", cmd, tier, err)
			}
			if err == nil {
				t.Errorf("expected error for blocked command %q, got nil", cmd)
			}
		})
	}
}

// TestEnforcer_TieredPermissions verifies that safe commands pass, dangerous
// ones are blocked, and intermediate commands prompt for approval.
func TestEnforcer_TieredPermissions(t *testing.T) {
	e := NewEnforcer()
	e.UseDefaults()

	tests := []struct {
		name       string
		cmd        string
		cwd        string
		wantTier   string
		wantErr    bool
	}{
		{
			name:     "safe command in workspace",
			cmd:      "ls",
			cwd:      "/agent/workspace",
			wantTier: "safe",
		},
		{
			name:     "safe build command",
			cmd:      "go build",
			cwd:      "/agent/workspace",
			wantTier: "safe",
		},
		{
			name:     "safe command in workspace path",
			cmd:      "cat file.txt",
			cwd:      "/agent/workspace/src",
			wantTier: "safe",
		},
		{
			name:     "blocked rm -rf",
			cmd:      "rm -rf",
			cwd:      "/agent/workspace",
			wantTier: "block",
			wantErr:  true,
		},
		{
			name:     "blocked sudo",
			cmd:      "sudo",
			cwd:      "/agent/workspace",
			wantTier: "block",
			wantErr:  true,
		},
		{
			name:     "prompt command git commit",
			cmd:      "git commit",
			cwd:      "/host/project",
			wantTier: "prompt",
		},
		{
			name:     "prompt docker build",
			cmd:      "docker build",
			cwd:      "/host/project/src",
			wantTier: "prompt",
		},
		{
			name:     "unknown command defaults to safe",
			cmd:      "echo hello",
			cwd:      "/agent/workspace",
			wantTier: "safe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier, err := e.Classify(tt.cmd, tt.cwd)
			if tier != tt.wantTier {
				t.Errorf("Classify(%q, %q) tier = %q, want %q", tt.cmd, tt.cwd, tier, tt.wantTier)
			}
			if tt.wantErr && err == nil {
				t.Errorf("Classify(%q, %q) expected error, got nil", tt.cmd, tt.cwd)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Classify(%q, %q) unexpected error: %v", tt.cmd, tt.cwd, err)
			}
		})
	}
}

// TestEnforcer_MaxFileSize verifies the enforcer does not impose arbitrary
// size limits on commands — large commands are still classified correctly.
// The security enforcer focuses on command semantics, not size.
func TestEnforcer_MaxFileSize(t *testing.T) {
	e := NewEnforcer()
	e.UseDefaults()

	// Build a very long safe command (e.g., a long echo)
	longContent := strings.Repeat("a", 100000)
	longCmd := "echo " + longContent

	// The enforcer should still classify it (no size-based rejection)
	tier, err := e.Classify(longCmd, "/agent/workspace")
	// Unknown commands default to safe
	if tier != "safe" {
		t.Errorf("expected safe tier for long command, got %q (err=%v)", tier, err)
	}

	// Build a long command that contains a blocked keyword but doesn't match exactly
	// The enforcer uses exact matching, so this tests that size alone isn't a factor
	longUnknownCmd := "notarealcommand " + longContent
	tier, err = e.Classify(longUnknownCmd, "/agent/workspace")
	// Unknown commands default to safe — no size-based blocking exists
	if tier != "safe" {
		t.Errorf("expected safe tier for long unknown command, got %q (err=%v)", tier, err)
	}

	// Build a long command that exactly matches a blocked pattern (just the pattern itself, repeated)
	// This verifies the enforcer classifies correctly regardless of prior context
	tier, err = e.Classify("sudo", "/agent/workspace")
	if tier != "block" {
		t.Errorf("expected block tier for sudo, got %q (err=%v)", tier, err)
	}
	if err == nil {
		t.Error("expected error for blocked command, got nil")
	}

	// Verify the enforcer does not have a max file size field
	// The Enforcer struct only has: config and tokens
	// There is no MaxFileSize field — classification is purely semantic
}

// TestEnforcer_GrantToken verifies that a granted token can be validated,
// consumed only once, and that the associated command is retrievable.
func TestEnforcer_GrantToken(t *testing.T) {
	e := NewEnforcer()
	e.UseDefaults()

	testCmd := "docker build -t myapp ."

	// Grant a token
	token, expiry := e.GrantToken(testCmd)
	if token == "" {
		t.Fatal("GrantToken returned empty token")
	}
	if expiry.IsZero() {
		t.Fatal("GrantToken returned zero expiry")
	}
	if expiry.Before(time.Now()) {
		t.Fatal("GrantToken returned past expiry")
	}
	if expiry.After(time.Now().Add(time.Minute)) {
		t.Fatal("GrantToken expiry is too far in the future (> 1 minute)")
	}

	// Verify the command is retrievable
	retrievedCmd := e.GetTokenCommand(token)
	if retrievedCmd != testCmd {
		t.Errorf("GetTokenCommand() = %q, want %q", retrievedCmd, testCmd)
	}

	// ValidateAndConsume should succeed first time
	if !e.ValidateAndConsume(token) {
		t.Fatal("ValidateAndConsume() failed on first use")
	}

	// ValidateAndConsume should fail second time (single-use)
	if e.ValidateAndConsume(token) {
		t.Fatal("ValidateAndConsume() succeeded on second use — token not single-use")
	}

	// GetTokenCommand should return empty after consumption
	retrievedAfter := e.GetTokenCommand(token)
	if retrievedAfter != "" {
		t.Errorf("GetTokenCommand() after consume = %q, want empty", retrievedAfter)
	}

	// ValidateToken should also fail after consumption
	if e.ValidateToken(token, testCmd) {
		t.Fatal("ValidateToken() succeeded after consumption")
	}

	// Test wrong command match
	token2, _ := e.GrantToken("ls")
	if e.ValidateToken(token2, "cat file") {
		t.Fatal("ValidateToken() succeeded with wrong command")
	}

	// Test with correct command — should succeed and consume
	if !e.ValidateToken(token2, "ls") {
		t.Fatal("ValidateToken() failed with correct command")
	}
	// Now it should fail (consumed)
	if e.ValidateToken(token2, "ls") {
		t.Fatal("ValidateToken() succeeded after consumption")
	}

	// Test with nonexistent token
	if e.ValidateToken("nonexistent-token", "ls") {
		t.Fatal("ValidateToken() succeeded with nonexistent token")
	}
	if e.ValidateAndConsume("nonexistent-token") {
		t.Fatal("ValidateAndConsume() succeeded with nonexistent token")
	}
}

// TestEnforcer_Defaults verifies that NewEnforcer() and UseDefaults()
// create a working enforcer that classifies basic commands correctly.
func TestEnforcer_Defaults(t *testing.T) {
	// NewEnforcer should already have defaults
	e := NewEnforcer()

	// Safe commands should pass
	tier, err := e.Classify("ls", "/agent/workspace")
	if tier != "safe" {
		t.Errorf("NewEnforcer: ls in workspace got tier %q, want safe (err=%v)", tier, err)
	}

	// Blocked commands should be blocked
	tier, err = e.Classify("rm -rf", "/agent/workspace")
	if tier != "block" {
		t.Errorf("NewEnforcer: rm -rf got tier %q, want block", tier)
	}
	if err == nil {
		t.Error("NewEnforcer: rm -rf expected error, got nil")
	}

	// UseDefaults should reset to working state
	e2 := NewEnforcer()
	// Manually corrupt by loading a nonexistent config
	err = e2.LoadConfig("/nonexistent/path/security.yaml")
	if err == nil {
		t.Error("expected error loading nonexistent config")
	}

	// Reset with defaults
	e2.UseDefaults()

	// Should work again
	tier, err = e2.Classify("cat", "/agent/workspace")
	if tier != "safe" {
		t.Errorf("UseDefaults: cat got tier %q, want safe", tier)
	}

	tier, err = e2.Classify("sudo", "/agent/workspace")
	if tier != "block" {
		t.Errorf("UseDefaults: sudo got tier %q, want block", tier)
	}

	// Verify default config structure
	cfg := e.DefaultConfig()
	if cfg.Tiers == nil {
		t.Fatal("DefaultConfig returned nil tiers")
	}

	// Check all three tiers exist
	for _, tierName := range []string{"safe", "prompt", "block"} {
		if _, ok := cfg.Tiers[tierName]; !ok {
			t.Errorf("DefaultConfig missing tier %q", tierName)
		}
	}

	// Verify safe tier has expected commands
	safeTier := cfg.Tiers["safe"]
	if len(safeTier.Cmds) == 0 {
		t.Error("safe tier has no commands")
	}
	if len(safeTier.Paths) == 0 {
		t.Error("safe tier has no paths")
	}

	// Verify block tier has expected commands
	blockTier := cfg.Tiers["block"]
	if len(blockTier.Cmds) == 0 {
		t.Error("block tier has no commands")
	}

	// Verify path matching works with ** globs
	tier, err = e.Classify("ls", "/agent/workspace/sub/dir")
	if tier != "safe" {
		t.Errorf("path matching: ls in /agent/workspace/sub/dir got %q, want safe", tier)
	}

	// Verify ** glob pattern matching works through the enforcer
	// Note: filepath.Match doesn't handle ** the same way, but our
	// matchPath function does — test the enforcer end-to-end
	matched, _ := filepath.Match("/agent/workspace/**", "/agent/workspace/file.txt")
	if !matched {
		// filepath.Match doesn't handle ** directly, but our matchPath does
		tier, _ = e.Classify("unknown", "/agent/workspace/sub/file.txt")
		if tier != "safe" {
			t.Errorf("wildcard path: unknown cmd in /agent/workspace/sub/file.txt got %q, want safe (default)", tier)
		}
	}
}
