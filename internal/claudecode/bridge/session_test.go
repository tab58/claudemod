package bridge

import (
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestSession_Wait(t *testing.T) {
	b := newTestBridge(t)
	defer b.Close()

	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	s, err := b.Spawn(catPath, nil, Config{})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := b.Activate(s); err != nil {
		t.Fatalf("activate: %v", err)
	}

	// Send EOF to terminate cat.
	_, _ = s.pty.Write([]byte{4})

	code, err := s.Wait()
	if err != nil {
		t.Fatalf("wait error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestSession_WaitNonZeroExit(t *testing.T) {
	b := newTestBridge(t)
	defer b.Close()

	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found in PATH")
	}

	s, err := b.Spawn(shPath, []string{"-c", "exit 42"}, Config{})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := b.Activate(s); err != nil {
		t.Fatalf("activate: %v", err)
	}

	code, err := s.Wait()
	if err != nil {
		t.Fatalf("wait error: %v", err)
	}
	if code != 42 {
		t.Errorf("expected exit code 42, got %d", code)
	}
}

func TestSession_Done(t *testing.T) {
	b := newTestBridge(t)
	defer b.Close()

	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found in PATH")
	}

	s, err := b.Spawn(shPath, []string{"-c", "exit 0"}, Config{})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := b.Activate(s); err != nil {
		t.Fatalf("activate: %v", err)
	}

	select {
	case <-s.Done():
		// expected
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Done channel")
	}
}

func TestSession_SuspendAndResume(t *testing.T) {
	b := newTestBridge(t)
	defer b.Close()

	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	s, err := b.Spawn(catPath, nil, Config{})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := b.Activate(s); err != nil {
		t.Fatalf("activate: %v", err)
	}

	if err := s.Suspend(); err != nil {
		t.Fatalf("suspend: %v", err)
	}

	if err := s.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}

	// After resume, send EOF and wait for clean exit.
	_, _ = s.pty.Write([]byte{4})

	code, err := s.Wait()
	if err != nil {
		t.Fatalf("wait error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0 after suspend/resume, got %d", code)
	}
}

func TestSession_PerSessionMiddleware(t *testing.T) {
	b := newTestBridge(t)
	defer b.Close()

	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	upper := func(data []byte) []byte {
		return []byte(strings.ToUpper(string(data)))
	}

	s, err := b.Spawn(catPath, nil, Config{ProcessOutput: upper})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := b.Activate(s); err != nil {
		t.Fatalf("activate: %v", err)
	}

	// Write input to cat's PTY; cat echoes it back through the output pump.
	_, _ = s.pty.Write([]byte("hello\n"))

	// Read from bridge stdout (the test buffer).
	buf := b.stdout.(*syncBuffer)
	waitForSyncOutput(t, buf, "HELLO", 3*time.Second)

	_, _ = s.pty.Write([]byte{4})
	_, _ = s.Wait()
}

// syncBuffer is a thread-safe bytes buffer for use as test stdout.
type syncBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (sb *syncBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.buf = append(sb.buf, p...)
	return len(p), nil
}

func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return string(sb.buf)
}

// newTestBridge creates a Bridge backed by a PTY pair so tests don't
// need a real terminal. The slave side satisfies raw-mode ioctls and the
// master side feeds stdin. Stdout goes to a syncBuffer.
func newTestBridge(t *testing.T) *Bridge {
	t.Helper()

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatalf("open test pty: %v", err)
	}
	t.Cleanup(func() {
		master.Close()
		slave.Close()
	})

	outBuf := &syncBuffer{}
	bridgeCfg := BridgeConfig{
		Stdin:  slave,
		Stdout: outBuf,
	}

	b, err := New(bridgeCfg)
	if err != nil {
		// If raw mode fails (e.g. CI without terminal), try the master side.
		var errM error
		bridgeCfg.Stdin = master
		b, errM = New(bridgeCfg)
		if errM != nil {
			t.Fatalf("new bridge: slave err=%v, master err=%v", err, errM)
		}
	}

	return b
}

// waitForSyncOutput polls the syncBuffer until it contains the expected string.
func waitForSyncOutput(t *testing.T, buf *syncBuffer, expected string, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %q; output so far: %q", expected, buf.String())
		default:
		}
		if strings.Contains(buf.String(), expected) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
