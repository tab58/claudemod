package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPhaseLog(t *testing.T) {
	tests := []struct {
		name      string
		content   string // file content; empty string means no file created
		noFile    bool   // if true, don't create the file at all
		wantLen   int
		wantErr   bool
		wantFirst PhaseLogEntry // checked only when wantLen > 0
	}{
		{
			name:    "missing file returns nil slice no error",
			noFile:  true,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "empty file returns nil slice no error",
			content: "",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "single entry",
			content: `{"timestamp":"2026-01-01T00:00:00Z","phase":"bootstrap","action":"advance","discussion_summary":"did stuff"}` + "\n",
			wantLen: 1,
			wantErr: false,
			wantFirst: PhaseLogEntry{
				Timestamp:         "2026-01-01T00:00:00Z",
				Phase:             "bootstrap",
				Action:            "advance",
				DiscussionSummary: "did stuff",
			},
		},
		{
			name: "multiple entries",
			content: `{"timestamp":"t1","phase":"bootstrap","action":"advance"}` + "\n" +
				`{"timestamp":"t2","phase":"discuss-feature","action":"advance"}` + "\n" +
				`{"timestamp":"t3","phase":"discuss-feature","action":"rollback","explanation":"missed reqs"}` + "\n",
			wantLen: 3,
			wantErr: false,
			wantFirst: PhaseLogEntry{
				Timestamp: "t1",
				Phase:     "bootstrap",
				Action:    "advance",
			},
		},
		{
			name:    "corrupt line returns error",
			content: `{"timestamp":"t1","phase":"bootstrap","action":"advance"}` + "\n" + "not json\n",
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "blank lines are skipped",
			content: "\n" + `{"timestamp":"t1","phase":"bootstrap","action":"advance"}` + "\n" + "\n",
			wantLen: 1,
			wantErr: false,
			wantFirst: PhaseLogEntry{
				Timestamp: "t1",
				Phase:     "bootstrap",
				Action:    "advance",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			app := &App{claudeModDir: dir}

			if !tt.noFile {
				err := os.WriteFile(filepath.Join(dir, phaseLogFileName), []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("setup: write test file: %v", err)
				}
			}

			entries, err := app.readPhaseLog()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(entries) != tt.wantLen {
				t.Fatalf("got %d entries, want %d", len(entries), tt.wantLen)
			}
			if tt.wantLen > 0 {
				got := entries[0]
				want := tt.wantFirst
				if got.Timestamp != want.Timestamp || got.Phase != want.Phase || got.Action != want.Action || got.DiscussionSummary != want.DiscussionSummary {
					t.Errorf("first entry = %+v, want %+v", got, want)
				}
			}
		})
	}
}

func TestAppendPhaseLog(t *testing.T) {
	dir := t.TempDir()
	app := &App{claudeModDir: dir}

	// First append creates the file
	entry1 := PhaseLogEntry{
		Timestamp: "t1",
		Phase:     "bootstrap",
		Action:    "advance",
	}
	if err := app.appendPhaseLog(entry1); err != nil {
		t.Fatalf("first append: %v", err)
	}

	// Second append adds to existing file
	entry2 := PhaseLogEntry{
		Timestamp:         "t2",
		Phase:             "discuss-feature",
		Action:            "advance",
		DiscussionSummary: "discussed requirements",
	}
	if err := app.appendPhaseLog(entry2); err != nil {
		t.Fatalf("second append: %v", err)
	}

	// Read back and verify
	entries, err := app.readPhaseLog()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Phase != "bootstrap" {
		t.Errorf("entries[0].Phase = %q, want %q", entries[0].Phase, "bootstrap")
	}
	if entries[1].Phase != "discuss-feature" {
		t.Errorf("entries[1].Phase = %q, want %q", entries[1].Phase, "discuss-feature")
	}
	if entries[1].DiscussionSummary != "discussed requirements" {
		t.Errorf("entries[1].DiscussionSummary = %q, want %q", entries[1].DiscussionSummary, "discussed requirements")
	}
}

func TestFormatPhaseLog(t *testing.T) {
	tests := []struct {
		name    string
		entries []PhaseLogEntry
		want    string
	}{
		{
			name:    "empty entries returns empty string",
			entries: nil,
			want:    "",
		},
		{
			name: "single entry",
			entries: []PhaseLogEntry{
				{Timestamp: "t1", Phase: "bootstrap", Action: "advance"},
			},
			want: "- [t1] Phase 'bootstrap': action=advance\n",
		},
		{
			name: "entry with all optional fields",
			entries: []PhaseLogEntry{
				{
					Timestamp:         "t1",
					Phase:             "scope-plan",
					Action:            "rollback",
					DiscussionSummary: "found issue",
					Explanation:       "missed req",
					Recommendation:    "redo specs",
				},
			},
			want: "- [t1] Phase 'scope-plan': action=rollback | summary: found issue | explanation: missed req | recommendation: redo specs\n",
		},
		{
			name: "multiple entries",
			entries: []PhaseLogEntry{
				{Timestamp: "t1", Phase: "bootstrap", Action: "advance"},
				{Timestamp: "t2", Phase: "discuss-feature", Action: "advance", DiscussionSummary: "good talk"},
			},
			want: "- [t1] Phase 'bootstrap': action=advance\n- [t2] Phase 'discuss-feature': action=advance | summary: good talk\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPhaseLog(tt.entries)
			if got != tt.want {
				t.Errorf("formatPhaseLog() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}
