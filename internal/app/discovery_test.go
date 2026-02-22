package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverProjects(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, root string)
		wantMulti      bool
		wantChildNames []string
		wantHasSpecs   map[string]bool // name -> HasSpecs
		wantErr        bool
	}{
		{
			name:           "empty directory",
			setup:          func(t *testing.T, root string) {},
			wantMulti:      false,
			wantChildNames: nil,
		},
		{
			name: "single child with specs",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, root, "service-auth/.claudemod/spec")
				writeFile(t, filepath.Join(root, "service-auth/.claudemod/spec/INDEX.md"), "# Auth")
			},
			wantMulti:      true,
			wantChildNames: []string{"service-auth"},
			wantHasSpecs:   map[string]bool{"service-auth": true},
		},
		{
			name: "single child without specs",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, root, "service-api/.claudemod")
			},
			wantMulti:      true,
			wantChildNames: []string{"service-api"},
			wantHasSpecs:   map[string]bool{"service-api": false},
		},
		{
			name: "multiple children sorted alphabetically",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, root, "zeta-service/.claudemod")
				mkdirAll(t, root, "alpha-service/.claudemod")
				mkdirAll(t, root, "middle-service/.claudemod")
			},
			wantMulti:      true,
			wantChildNames: []string{"alpha-service", "middle-service", "zeta-service"},
			wantHasSpecs:   map[string]bool{"alpha-service": false, "middle-service": false, "zeta-service": false},
		},
		{
			name: "mixed children with and without claudemod",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, root, "has-claudemod/.claudemod")
				mkdirAll(t, root, "no-claudemod/src")
				mkdirAll(t, root, "also-has-claudemod/.claudemod/spec")
				writeFile(t, filepath.Join(root, "also-has-claudemod/.claudemod/spec/INDEX.md"), "# Spec")
			},
			wantMulti:      true,
			wantChildNames: []string{"also-has-claudemod", "has-claudemod"},
			wantHasSpecs:   map[string]bool{"also-has-claudemod": true, "has-claudemod": false},
		},
		{
			name: "hidden directories are skipped",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, root, ".hidden-project/.claudemod")
				mkdirAll(t, root, "visible-project/.claudemod")
			},
			wantMulti:      true,
			wantChildNames: []string{"visible-project"},
			wantHasSpecs:   map[string]bool{"visible-project": false},
		},
		{
			name: "skip-list directories are skipped",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, root, "node_modules/some-package/.claudemod")
				mkdirAll(t, root, "vendor/dep/.claudemod")
				mkdirAll(t, root, "real-project/.claudemod")
			},
			wantMulti:      true,
			wantChildNames: []string{"real-project"},
			wantHasSpecs:   map[string]bool{"real-project": false},
		},
		{
			name: "parent own claudemod is excluded",
			setup: func(t *testing.T, root string) {
				// parent's .claudemod should not appear as a child
				mkdirAll(t, root, ".claudemod")
			},
			wantMulti:      false,
			wantChildNames: nil,
		},
		{
			name: "nested claudemod does not descend further",
			setup: func(t *testing.T, root string) {
				// child has its own child — only the first level should be found
				mkdirAll(t, root, "outer/.claudemod")
				mkdirAll(t, root, "outer/inner/.claudemod")
			},
			wantMulti:      true,
			wantChildNames: []string{"outer"},
			wantHasSpecs:   map[string]bool{"outer": false},
		},
		{
			name: "child in subdirectory",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, root, "services/auth/.claudemod/spec")
				writeFile(t, filepath.Join(root, "services/auth/.claudemod/spec/INDEX.md"), "# Auth")
			},
			wantMulti:      true,
			wantChildNames: []string{"auth"},
			wantHasSpecs:   map[string]bool{"auth": true},
		},
		{
			name: "has specs false when spec dir exists but no INDEX.md",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, root, "svc/.claudemod/spec")
				writeFile(t, filepath.Join(root, "svc/.claudemod/spec/other.md"), "# Other")
			},
			wantMulti:      true,
			wantChildNames: []string{"svc"},
			wantHasSpecs:   map[string]bool{"svc": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			layout, err := DiscoverProjects(root)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DiscoverProjects() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if layout.IsMultiProject != tt.wantMulti {
				t.Errorf("IsMultiProject = %v, want %v", layout.IsMultiProject, tt.wantMulti)
			}

			if layout.ParentDir != root {
				t.Errorf("ParentDir = %q, want %q", layout.ParentDir, root)
			}

			gotNames := make([]string, len(layout.ChildProjects))
			for i, c := range layout.ChildProjects {
				gotNames[i] = c.Name
			}

			if len(gotNames) != len(tt.wantChildNames) {
				t.Fatalf("got %d children %v, want %d %v", len(gotNames), gotNames, len(tt.wantChildNames), tt.wantChildNames)
			}
			for i := range gotNames {
				if gotNames[i] != tt.wantChildNames[i] {
					t.Errorf("child[%d].Name = %q, want %q", i, gotNames[i], tt.wantChildNames[i])
				}
			}

			if tt.wantHasSpecs != nil {
				for _, c := range layout.ChildProjects {
					want, ok := tt.wantHasSpecs[c.Name]
					if !ok {
						continue
					}
					if c.HasSpecs != want {
						t.Errorf("child %q HasSpecs = %v, want %v", c.Name, c.HasSpecs, want)
					}
				}
			}
		})
	}
}

func TestComputeAdditionalDirs(t *testing.T) {
	multiLayout := ProjectLayout{
		IsMultiProject: true,
		ParentDir:      "/root",
		ChildProjects: []ChildProject{
			{Name: "svc-a", Path: "/root/svc-a"},
			{Name: "svc-b", Path: "/root/svc-b"},
			{Name: "svc-c", Path: "/root/svc-c"},
		},
	}

	singleLayout := ProjectLayout{
		IsMultiProject: false,
		ParentDir:      "/root",
	}

	tests := []struct {
		name          string
		layout        ProjectLayout
		phaseName     string
		affectedRepos []string
		want          []string
	}{
		{
			name:      "single project returns nil",
			layout:    singleLayout,
			phaseName: "tdd-red",
			want:      nil,
		},
		{
			name:      "planning phase gets all children",
			layout:    multiLayout,
			phaseName: "discuss-feature",
			want:      []string{"/root/svc-a", "/root/svc-b", "/root/svc-c"},
		},
		{
			name:      "bootstrap is a planning phase",
			layout:    multiLayout,
			phaseName: "bootstrap",
			want:      []string{"/root/svc-a", "/root/svc-b", "/root/svc-c"},
		},
		{
			name:      "scope-plan is a planning phase",
			layout:    multiLayout,
			phaseName: "scope-plan",
			want:      []string{"/root/svc-a", "/root/svc-b", "/root/svc-c"},
		},
		{
			name:      "describe-bug is a planning phase",
			layout:    multiLayout,
			phaseName: "describe-bug",
			want:      []string{"/root/svc-a", "/root/svc-b", "/root/svc-c"},
		},
		{
			name:          "impl phase narrows to affected repos",
			layout:        multiLayout,
			phaseName:     "tdd-red",
			affectedRepos: []string{"svc-a", "svc-c"},
			want:          []string{"/root/svc-a", "/root/svc-c"},
		},
		{
			name:          "impl phase with single affected repo",
			layout:        multiLayout,
			phaseName:     "tdd-green",
			affectedRepos: []string{"svc-b"},
			want:          []string{"/root/svc-b"},
		},
		{
			name:          "impl phase with unrecognized repo returns empty",
			layout:        multiLayout,
			phaseName:     "code-review",
			affectedRepos: []string{"svc-unknown"},
			want:          []string{},
		},
		{
			name:      "impl phase with no affected repos falls back to all",
			layout:    multiLayout,
			phaseName: "tdd-red",
			want:      []string{"/root/svc-a", "/root/svc-b", "/root/svc-c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeAdditionalDirs(tt.layout, tt.phaseName, tt.affectedRepos)

			if tt.want == nil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %v (len %d), want %v (len %d)", got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// mkdirAll creates a directory hierarchy relative to root.
func mkdirAll(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, rel), 0755); err != nil {
		t.Fatal(err)
	}
}

// writeFile creates a file with content, creating parent dirs as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
