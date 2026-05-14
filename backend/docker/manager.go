package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Manager provides Docker container management via the Docker CLI.
type Manager struct {
	cli string // path to docker CLI, defaults to "docker"
}

// ContainerInfo represents essential info about a container.
type ContainerInfo struct {
	ID      string `json:"id"`
	Names   string `json:"names"`
	Image   string `json:"image"`
	State   string `json:"state"`
	Status  string `json:"status"`
	Created int64  `json:"created"`
}

// ContainerStats holds live resource stats for a container.
type ContainerStats struct {
	ID      string                 `json:"id"`
	CPU     map[string]interface{} `json:"cpu"`
	Memory  map[string]interface{} `json:"memory"`
	Network map[string]interface{} `json:"network"`
}

// NewManager creates a Docker CLI manager.
func NewManager() (*Manager, error) {
	return &Manager{cli: "docker"}, nil
}

// ListContainers returns a list of all containers (running and stopped).
func (m *Manager) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, m.cli, "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.State}}\t{{.Status}}\t{{.Created}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []ContainerInfo
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			continue
		}
		created, _ := strconv.ParseInt(fields[5], 10, 64)
		result = append(result, ContainerInfo{
			ID:      fields[0],
			Names:   fields[1],
			Image:   fields[2],
			State:   fields[3],
			Status:  fields[4],
			Created: created,
		})
	}
	return result, nil
}

// GetStats returns live stats for a specific container using docker stats.
func (m *Manager) GetStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	cmd := exec.CommandContext(ctx, m.cli, "stats", containerID, "--no-stream", "--format", "{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get stats for %s: %w", containerID, err)
	}

	output := strings.TrimSpace(out.String())
	fields := strings.Split(output, "\t")
	if len(fields) < 3 {
		return nil, fmt.Errorf("unexpected stats output format")
	}

	// Parse CPU percentage
	cpuPercent := strings.TrimSuffix(fields[0], "%")
	cpuVal, _ := strconv.ParseFloat(cpuPercent, 64)

	// Parse memory: "1.5GiB / 8GiB" -> used, limit
	memParts := strings.Split(fields[1], " / ")
	var memUsed, memLimit float64
	var memUsedStr, memLimitStr string
	if len(memParts) == 2 {
		memUsedStr = strings.TrimSpace(memParts[0])
		memLimitStr = strings.TrimSpace(memParts[1])
		// Simple parse - just convert to bytes assuming GiB/MiB suffix
		memUsed = parseMemory(memUsedStr)
		memLimit = parseMemory(memLimitStr)
	}

	// Parse network I/O: "1.5MB / 2MB" -> rx, tx
	netParts := strings.Split(fields[2], " / ")
	var netRx, netTx float64
	if len(netParts) == 2 {
		netRx = parseNetwork(netParts[0])
		netTx = parseNetwork(netParts[1])
	}

	cpu := make(map[string]interface{})
	cpu["percent"] = cpuVal

	mem := make(map[string]interface{})
	mem["usage"] = memUsed
	mem["limit"] = memLimit
	if memLimit > 0 {
		mem["percent"] = memUsed / memLimit * 100
	}

	net := make(map[string]interface{})
	net["rx_bytes"] = netRx
	net["tx_bytes"] = netTx

	return &ContainerStats{
		ID:      containerID,
		CPU:     cpu,
		Memory:  mem,
		Network: net,
	}, nil
}

// parseMemory parses memory strings like "1.5GiB" or "500MiB" to bytes.
func parseMemory(s string) float64 {
	s = strings.TrimSpace(s)
	var multiplier float64 = 1
	if strings.HasSuffix(s, "GiB") || strings.HasSuffix(s, "GB") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GiB")
		s = strings.TrimSuffix(s, "GB")
	} else if strings.HasSuffix(s, "MiB") || strings.HasSuffix(s, "MB") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MiB")
		s = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "KiB") || strings.HasSuffix(s, "KB") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "KiB")
		s = strings.TrimSuffix(s, "KB")
	}
	val, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return val * multiplier
}

// parseNetwork parses network I/O strings like "1.5MB" to bytes.
func parseNetwork(s string) float64 {
	s = strings.TrimSpace(s)
	var multiplier float64 = 1
	if strings.HasSuffix(s, "GB") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "KB") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "KB")
	} else if strings.HasSuffix(s, "B") {
		s = strings.TrimSuffix(s, "B")
	}
	val, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return val * multiplier
}

// StartContainer starts a stopped or created container.
func (m *Manager) StartContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, m.cli, "start", containerID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}
	return nil
}

// StopContainer stops a running container.
func (m *Manager) StopContainer(ctx context.Context, containerID string, timeout *int) error {
	args := []string{"stop", containerID}
	if timeout != nil {
		args = append(args, "-t", strconv.Itoa(*timeout))
	}
	cmd := exec.CommandContext(ctx, m.cli, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

// RestartContainer restarts a container.
func (m *Manager) RestartContainer(ctx context.Context, containerID string, timeout *int) error {
	args := []string{"restart", containerID}
	if timeout != nil {
		args = append(args, "-t", strconv.Itoa(*timeout))
	}
	cmd := exec.CommandContext(ctx, m.cli, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerID, err)
	}
	return nil
}

// RemoveContainer removes a container (must be stopped unless force=true).
func (m *Manager) RemoveContainer(ctx context.Context, containerID string, force, removeVolumes bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	if removeVolumes {
		args = append(args, "-v")
	}
	args = append(args, containerID)
	cmd := exec.CommandContext(ctx, m.cli, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}
	return nil
}

// GetLogs returns the stdout/stderr logs of a container.
func (m *Manager) GetLogs(ctx context.Context, containerID string, tail string) (string, error) {
	if tail == "" {
		tail = "100"
	}
	cmd := exec.CommandContext(ctx, m.cli, "logs", "--tail", tail, containerID)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		// docker logs may return error even with output, check stderr
		if errOut.Len() > 0 && out.Len() == 0 {
			return "", fmt.Errorf("failed to get logs for %s: %w", containerID, err)
		}
	}
	// Combine stdout and stderr
	logs := out.String()
	if errOut.Len() > 0 {
		logs = logs + "\n" + errOut.String()
	}
	return logs, nil
}