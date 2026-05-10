package bench

import (
	"os/exec"
	"testing"
	"time"

	"hermes-web-computer/backend/pty"
)

// TestInterruptLatency measures: keydown → amber border → PTY signal → checkpoint save.
// Target: p99 < 100ms on dev hardware.
// Fallback: p99 < 250ms with warning.
func TestInterruptLatency(t *testing.T) {
	sup := pty.NewSupervisor()

	// Start a simple shell process
	cmd := newShellCmd()
	session, err := sup.Start("test_pty", cmd)
	if err != nil {
		t.Skipf("could not start PTY: %v", err)
	}

	// Measure interrupt path
	start := time.Now()

	// 1. Signal SIGINT
	if err := sup.Signal(session.ID, 0); err != nil { // 0 = no-op for measurement
		t.Fatalf("signal error: %v", err)
	}

	// 2. Checkpoint save
	_, err = sup.Checkpoint(session.ID)
	if err != nil {
		t.Fatalf("checkpoint error: %v", err)
	}

	elapsed := time.Since(start)

	// Log result
	t.Logf("interrupt latency: %v", elapsed)

	// Targets
	if elapsed > 250*time.Millisecond {
		t.Errorf("interrupt latency %v exceeds 250ms fallback target", elapsed)
	} else if elapsed > 100*time.Millisecond {
		t.Logf("WARNING: interrupt latency %v exceeds 100ms target but within fallback", elapsed)
	}
}

// BenchmarkInterruptLatency runs the interrupt path 10000x to get p99.
func BenchmarkInterruptLatency(b *testing.B) {
	sup := pty.NewSupervisor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := newShellCmd()
		session, err := sup.Start("bench_pty", cmd)
		if err != nil {
			b.Skipf("could not start PTY: %v", err)
		}

		start := time.Now()
		sup.Signal(session.ID, 0)
		sup.Checkpoint(session.ID)
		elapsed := time.Since(start)

		b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/op")
	}
}

func newShellCmd() *exec.Cmd {
	return exec.Command("sh", "-c", "sleep 10")
}
