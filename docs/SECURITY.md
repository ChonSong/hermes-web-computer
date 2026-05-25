# Security Model — Hermes Web Computer

## Overview

HWC uses a tiered permission system to classify and control command execution based on working directory and command patterns. The system supports three tiers: **safe**, **prompt**, and **block**.

## Tiers

### Safe Tier
- **Purpose**: Commands that are read-only, non-destructive, and low-risk
- **Paths**: `/agent/workspace/**` (and subdirectories)
- **Commands**: `ls`, `cat`, `git status`, `pip install`, `npm install`, `go build`, `make`
- **Behavior**: Executed immediately without user confirmation

### Prompt Tier
- **Purpose**: Commands that modify state or take longer to complete
- **Paths**: `/host/project/**`
- **Commands**: `git commit`, `npm run build`, `docker build`
- **Behavior**: Generates a time-limited token (30s) requiring user approval via the UI

### Block Tier
- **Purpose**: Commands that are destructive, privileged, or could compromise system integrity
- **Commands**: `rm -rf`, `chmod 777`, `curl | bash`, `sudo`, `dd`, `mkfs`, `mkfs.*`, `shutdown`, `reboot`, `kill -9`
- **Behavior**: Always blocked regardless of path; returns an error

## Token-Gated Execution

Commands classified as "prompt" tier require user approval before execution:

1. Agent attempts to run a prompt-tier command
2. Server generates a unique token (UUID) with 30-second expiry
3. UI displays an approval prompt to the user
4. User grants or denies via the `approval.grant` method
5. If granted, the token is consumed and the command is executed once

## Configuration File

The security rules are loaded from:
```
~/.hermes/hermes-web-computer/security.yaml
```

This can be overridden via the `HWC_SECURITY_CONFIG` environment variable.

### Example `security.yaml`

```yaml
tiers:
  safe:
    paths:
      - "/agent/workspace/**"
    cmds:
      - "ls"
      - "cat"
      - "git status"
  prompt:
    paths:
      - "/host/project/**"
    cmds:
      - "git commit"
      - "npm run build"
  block:
    cmds:
      - "rm -rf"
      - "sudo"
      - "dd"
      - "shutdown"
```

## Defaults

If no security config is found, HWC uses safe defaults that:
- Allow read-only commands in `/agent/workspace/**`
- Require approval for project-modifying commands
- Block destructive operations like `rm -rf`, `chmod 777`, `curl | bash`, `sudo`, `dd`, `mkfs`, `shutdown`, `reboot`, `kill -9`

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HWC_SECURITY_CONFIG` | Path to security config YAML | `~/.hermes/hermes-web-computer/security.yaml` |
| `HWC_STATE_DIR` | Base directory for sessions and config | `~/.hermes/hermes-web-computer` |