package bridge

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

// TestPTYRoundtrip validates the PTY bridge works by spawning "cat"
// (which echoes stdin to stdout) and verifying I/O roundtrip.
func TestPTYRoundtrip(t *testing.T) {
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	cmd := exec.Command(catPath)
	cmd.Env = cleanEnv(os.Environ())

	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("start pty: %v", err)
	}
	defer ptmx.Close()

	// Write test data to the PTY master (simulating user input).
	testData := "hello from test\n"
	_, err = ptmx.Write([]byte(testData))
	if err != nil {
		t.Fatalf("write to pty: %v", err)
	}

	// Read back from PTY master (cat echoes back through the PTY).
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
			if strings.Contains(collected.String(), "hello from test") {
				goto done
			}
		}
		if err != nil {
			// Timeout is expected, keep trying until deadline.
			continue
		}
	}

done:
	// Send EOF to terminate cat.
	ptmx.Write([]byte{4}) // Ctrl-D
	cmd.Wait()

	output := collected.String()
	if !strings.Contains(output, "hello from test") {
		t.Errorf("expected output to contain 'hello from test', got %q", output)
	}
}

// TestPTYWithTransform validates that the transform callback works.
func TestPTYWithTransform(t *testing.T) {
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found in PATH")
	}

	cmd := exec.Command(catPath)
	cmd.Env = cleanEnv(os.Environ())

	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("start pty: %v", err)
	}
	defer ptmx.Close()

	// Use an uppercasing transform on the "input" side.
	transform := func(data []byte) []byte {
		return []byte(strings.ToUpper(string(data)))
	}

	// Write transformed data.
	input := []byte("hello\n")
	transformed := transform(input)
	_, err = ptmx.Write(transformed)
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
			if strings.Contains(collected.String(), "HELLO") {
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
	if !strings.Contains(output, "HELLO") {
		t.Errorf("expected output to contain 'HELLO', got %q", output)
	}
}
