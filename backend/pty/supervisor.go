package pty

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// Supervisor manages PTY processes with ring buffer and checkpoint support.
type Supervisor struct {
	mu   sync.RWMutex
	ptys map[string]*PTYSession
}

// PTYSession wraps a PTY with metadata.
type PTYSession struct {
	ID      string
	Cmd     *exec.Cmd
	PTTY    *os.File
	RingBuf *RingBuffer // 1MB ring buffer for checkpoint
	Output  chan []byte // Channel for forwarding PTY output to clients
	mu      sync.Mutex
}

// RingBuffer is a fixed-size circular buffer for PTY output.
type RingBuffer struct {
	data  []byte
	start int
	end   int
	size  int
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

func (rb *RingBuffer) Write(p []byte) (int, error) {
	for _, b := range p {
		rb.data[rb.end] = b
		rb.end = (rb.end + 1) % rb.size
		if rb.end == rb.start {
			rb.start = (rb.start + 1) % rb.size
		}
	}
	return len(p), nil
}

func (rb *RingBuffer) ReadAll() []byte {
	result := make([]byte, 0, rb.size)
	if rb.end >= rb.start {
		result = append(result, rb.data[rb.start:rb.end]...)
	} else {
		result = append(result, rb.data[rb.start:]...)
		result = append(result, rb.data[:rb.end]...)
	}
	return result
}

func NewSupervisor() *Supervisor {
	return &Supervisor{
		ptys: make(map[string]*PTYSession),
	}
}

func (s *Supervisor) Start(id string, cmd *exec.Cmd) (*PTYSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	session := &PTYSession{
		ID:      id,
		Cmd:     cmd,
		PTTY:    p,
		RingBuf: NewRingBuffer(1024 * 1024), // 1MB
		Output:  make(chan []byte, 64),
	}
	s.ptys[id] = session

	// Start reading from PTY into ring buffer AND output channel
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := p.Read(buf)
			if err != nil {
				if err == io.EOF {
					return
				}
				return
			}
			session.mu.Lock()
			session.RingBuf.Write(buf[:n])
			session.mu.Unlock()
			// Forward to output channel (non-blocking)
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case session.Output <- data:
			default:
				// Drop if channel full
			}
		}
	}()

	return session, nil
}

func (s *Supervisor) Signal(id string, sig syscall.Signal) error {
	s.mu.RLock()
	session, ok := s.ptys[id]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	return syscall.Kill(-session.Cmd.Process.Pid, sig)
}

func (s *Supervisor) Checkpoint(id string) ([]byte, error) {
	s.mu.RLock()
	session, ok := s.ptys[id]
	s.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.RingBuf.ReadAll(), nil
}

// PTTY returns the PTY file descriptor for writing.
func (s *Supervisor) PTY(id string) *os.File {
	s.mu.RLock()
	session, ok := s.ptys[id]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	return session.PTTY
}
