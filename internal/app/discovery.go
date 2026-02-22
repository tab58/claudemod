package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChildProject describes a child repository discovered within the workspace.
type ChildProject struct {
	Name    string // directory basename (e.g., "service-auth")
	Path    string // absolute path to child project dir
	SpecDir string // absolute path to .claudemod/spec/
	HasSpecs bool  // true if spec/INDEX.md exists
}

// ProjectLayout captures the multi-project topology of a workspace.
type ProjectLayout struct {
	IsMultiProject bool
	ParentDir      string
	ChildProjects  []ChildProject // sorted by Name
}

// skipDirs lists directory basenames that should never be descended into.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	"target":       true,
	"build":        true,
	"dist":         true,
	"bin":          true,
}

// DiscoverProjects walks wd looking for child directories that contain a
// .claudemod/ folder. The parent's own .claudemod/ is excluded. Hidden
// directories (dot-prefixed) and common build/dependency directories are
// skipped.
func DiscoverProjects(wd string) (ProjectLayout, error) {
	absWd, err := filepath.Abs(wd)
	if err != nil {
		return ProjectLayout{}, fmt.Errorf("resolve workspace path: %w", err)
	}

	var children []ChildProject

	err = filepath.WalkDir(absWd, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip inaccessible entries
		}

		if !d.IsDir() {
			return nil
		}

		// skip the root itself
		if path == absWd {
			return nil
		}

		name := d.Name()

		// skip hidden directories
		if strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}

		// skip well-known non-project directories
		if skipDirs[name] {
			return filepath.SkipDir
		}

		// check for .claudemod/ inside this directory
		claudeModPath := filepath.Join(path, ".claudemod")
		info, statErr := os.Stat(claudeModPath)
		if statErr != nil || !info.IsDir() {
			return nil // not a project root, keep walking
		}

		// found a child project — record it and stop descending
		specDir := filepath.Join(claudeModPath, "spec")
		hasSpecs := fileExists(filepath.Join(specDir, "INDEX.md"))

		children = append(children, ChildProject{
			Name:    filepath.Base(path),
			Path:    path,
			SpecDir: specDir,
			HasSpecs: hasSpecs,
		})

		return filepath.SkipDir
	})
	if err != nil {
		return ProjectLayout{}, fmt.Errorf("walk workspace: %w", err)
	}

	// deterministic ordering
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})

	return ProjectLayout{
		IsMultiProject: len(children) > 0,
		ParentDir:      absWd,
		ChildProjects:  children,
	}, nil
}
