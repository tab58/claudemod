package bridge

import (
	"testing"
)

func TestCleanEnv_StripsClaudeVars(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"CLAUDECODE=1",
		"CLAUDE_CODE_SSE_PORT=12345",
		"CLAUDE_CODE_ENTRYPOINT=node",
		"PATH=/usr/bin",
		"SHELL=/bin/zsh",
	}

	cleaned := cleanEnv(env)

	expected := map[string]bool{
		"HOME=/home/user": true,
		"PATH=/usr/bin":   true,
		"SHELL=/bin/zsh":  true,
	}

	if len(cleaned) != len(expected) {
		t.Fatalf("expected %d env vars, got %d: %v", len(expected), len(cleaned), cleaned)
	}

	for _, e := range cleaned {
		if !expected[e] {
			t.Errorf("unexpected env var: %q", e)
		}
	}
}

func TestCleanEnv_PreservesNonClaudeVars(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"EDITOR=vim",
		"CLAUDE_UNRELATED=yes",
	}

	cleaned := cleanEnv(env)

	// CLAUDE_UNRELATED should be kept (not in stripped list)
	if len(cleaned) != 3 {
		t.Fatalf("expected 3 env vars, got %d: %v", len(cleaned), cleaned)
	}
}

func TestCleanEnv_EmptyEnv(t *testing.T) {
	cleaned := cleanEnv(nil)
	if len(cleaned) != 0 {
		t.Errorf("expected empty result, got %v", cleaned)
	}
}
