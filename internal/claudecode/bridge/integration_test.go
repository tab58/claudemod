package bridge

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

// TestRun_BackwardCompat validates the backward-compatible Run() wrapper
// works with a simple echo round-trip through cat.
func TestRun_BackwardCompat(t *testing.T) {
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	// We can't easily test Run() directly (it takes over the terminal),
	// so we test the low-level PTY round-trip that Run() relies on.
	cmd := exec.Command(catPath)
	cmd.Env = cleanEnv(cmd.Environ())

	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("start pty: %v", err)
	}
	defer ptmx.Close()

	testData := "hello from compat test\n"
	_, err = ptmx.Write([]byte(testData))
	if err != nil {
		t.Fatalf("write to pty: %v", err)
	}

	buf := make([]byte, 1024)
	var collected bytes.Buffer
	deadline := time.After(2 * time.Second)

	for {
		select {
		case <-deadline:
			goto done
		default:
		}

		ptmx.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, err := ptmx.Read(buf)
		if n > 0 {
			collected.Write(buf[:n])
			if strings.Contains(collected.String(), "hello from compat test") {
				goto done
			}
		}
		if err != nil {
			continue
		}
	}

done:
	ptmx.Write([]byte{4})
	cmd.Wait()

	output := collected.String()
	if !strings.Contains(output, "hello from compat test") {
		t.Errorf("expected output to contain 'hello from compat test', got %q", output)
	}
}

// TestMultiSessionWorkflow exercises the full spawn/activate/suspend/resume/switch cycle.
func TestMultiSessionWorkflow(t *testing.T) {
	b := newTestBridge(t)
	defer b.Close()

	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	// Spawn two sessions.
	s1, err := b.Spawn(catPath, nil, Config{})
	if err != nil {
		t.Fatalf("spawn s1: %v", err)
	}

	s2, err := b.Spawn(catPath, nil, Config{})
	if err != nil {
		t.Fatalf("spawn s2: %v", err)
	}

	buf := b.stdout.(*syncBuffer)

	// Activate s1, verify output.
	if err := b.Activate(s1); err != nil {
		t.Fatalf("activate s1: %v", err)
	}
	_, _ = s1.pty.Write([]byte("session-one\n"))
	waitForSyncOutput(t, buf, "session-one", 3*time.Second)

	// Suspend s1.
	if err := s1.Suspend(); err != nil {
		t.Fatalf("suspend s1: %v", err)
	}

	// Switch to s2.
	if err := b.Activate(s2); err != nil {
		t.Fatalf("activate s2: %v", err)
	}
	_, _ = s2.pty.Write([]byte("session-two\n"))
	waitForSyncOutput(t, buf, "session-two", 3*time.Second)

	// Suspend s2, switch back to s1.
	if err := s2.Suspend(); err != nil {
		t.Fatalf("suspend s2: %v", err)
	}
	if err := b.Activate(s1); err != nil {
		t.Fatalf("re-activate s1: %v", err)
	}
	if err := s1.Resume(); err != nil {
		t.Fatalf("resume s1: %v", err)
	}

	_, _ = s1.pty.Write([]byte("back-to-one\n"))
	waitForSyncOutput(t, buf, "back-to-one", 3*time.Second)

	// Clean up both sessions.
	_, _ = s1.pty.Write([]byte{4})
	s2.Resume() // must resume before EOF
	_, _ = s2.pty.Write([]byte{4})

	code1, err := s1.Wait()
	if err != nil {
		t.Fatalf("s1 wait error: %v", err)
	}
	if code1 != 0 {
		t.Errorf("s1 exit code: %d, expected 0", code1)
	}

	code2, err := s2.Wait()
	if err != nil {
		t.Fatalf("s2 wait error: %v", err)
	}
	if code2 != 0 {
		t.Errorf("s2 exit code: %d, expected 0", code2)
	}
}
