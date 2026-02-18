package bridge

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestBridge_NewAndClose(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatalf("open pty: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	buf := &syncBuffer{}
	b, err := New(BridgeConfig{Stdin: slave, Stdout: buf})
	if err != nil {
		t.Fatalf("new bridge: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Double close should be safe.
	if err := b.Close(); err != nil {
		t.Fatalf("double close: %v", err)
	}
}

func TestBridge_SpawnCreatesSession(t *testing.T) {
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
	if s == nil {
		t.Fatal("expected non-nil session")
	}

	// Clean up.
	_, _ = s.pty.Write([]byte{4})
	_, _ = s.Wait()
}

func TestBridge_ActivateSwitchesIO(t *testing.T) {
	b := newTestBridge(t)
	defer b.Close()

	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	s1, err := b.Spawn(catPath, nil, Config{})
	if err != nil {
		t.Fatalf("spawn s1: %v", err)
	}

	s2, err := b.Spawn(catPath, nil, Config{})
	if err != nil {
		t.Fatalf("spawn s2: %v", err)
	}

	// Activate s1, write through it.
	if err := b.Activate(s1); err != nil {
		t.Fatalf("activate s1: %v", err)
	}

	_, _ = s1.pty.Write([]byte("from-s1\n"))

	buf := b.stdout.(*syncBuffer)
	waitForSyncOutput(t, buf, "from-s1", 3*time.Second)

	// Switch to s2.
	if err := b.Activate(s2); err != nil {
		t.Fatalf("activate s2: %v", err)
	}

	_, _ = s2.pty.Write([]byte("from-s2\n"))
	waitForSyncOutput(t, buf, "from-s2", 3*time.Second)

	// Clean up.
	_, _ = s1.pty.Write([]byte{4})
	_, _ = s2.pty.Write([]byte{4})
	_, _ = s1.Wait()
	_, _ = s2.Wait()
}

func TestBridge_ActivateNilDiscards(t *testing.T) {
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

	// Deactivate all.
	if err := b.Activate(nil); err != nil {
		t.Fatalf("activate nil: %v", err)
	}

	if active := b.Active(); active != nil {
		t.Errorf("expected nil active, got %v", active)
	}

	_, _ = s.pty.Write([]byte{4})
	_, _ = s.Wait()
}

func TestBridge_SpawnAfterCloseErrors(t *testing.T) {
	b := newTestBridge(t)
	b.Close()

	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	_, err = b.Spawn(catPath, nil, Config{})
	if err == nil {
		t.Fatal("expected error spawning on closed bridge")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("expected 'closed' in error, got: %v", err)
	}
}
