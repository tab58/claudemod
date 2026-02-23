package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name  string
		check func(t *testing.T, info Info)
	}{
		{
			name: "default version is dev",
			check: func(t *testing.T, info Info) {
				if info.Version != "dev" {
					t.Errorf("Version = %q, want %q", info.Version, "dev")
				}
			},
		},
		{
			name: "default commit is none",
			check: func(t *testing.T, info Info) {
				if info.Commit != "none" {
					t.Errorf("Commit = %q, want %q", info.Commit, "none")
				}
			},
		},
		{
			name: "default date is unknown",
			check: func(t *testing.T, info Info) {
				if info.Date != "unknown" {
					t.Errorf("Date = %q, want %q", info.Date, "unknown")
				}
			},
		},
		{
			name: "Go version from runtime",
			check: func(t *testing.T, info Info) {
				if info.Go != runtime.Version() {
					t.Errorf("Go = %q, want %q", info.Go, runtime.Version())
				}
			},
		},
		{
			name: "OS from runtime",
			check: func(t *testing.T, info Info) {
				if info.OS != runtime.GOOS {
					t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
				}
			},
		},
		{
			name: "Arch from runtime",
			check: func(t *testing.T, info Info) {
				if info.Arch != runtime.GOARCH {
					t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := Get()
			tt.check(t, info)
		})
	}
}

func TestInfoString(t *testing.T) {
	info := Info{
		Version: "1.2.3",
		Commit:  "abc1234",
		Date:    "2026-02-22T00:00:00Z",
		Go:      "go1.24.5",
		OS:      "darwin",
		Arch:    "arm64",
	}

	got := info.String()
	want := "claudemod 1.2.3 (commit: abc1234, built: 2026-02-22T00:00:00Z, go1.24.5, darwin/arm64)"

	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestInfoStringContainsAllFields(t *testing.T) {
	info := Get()
	s := info.String()

	fields := []string{
		info.Version,
		info.Commit,
		info.Date,
		info.Go,
		info.OS,
		info.Arch,
	}

	for _, field := range fields {
		if !strings.Contains(s, field) {
			t.Errorf("String() = %q, missing field %q", s, field)
		}
	}
}
