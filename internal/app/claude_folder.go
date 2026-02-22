package app

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

//go:embed refs/*.md
var refsFS embed.FS

func SetupPluginFolder(wd string) error {
	// ensure the right folders exist
	err := ensureClaudeFolderExists(wd)
	if err != nil {
		return err
	}
	err = ensureClaudeModFolderExists(wd)
	if err != nil {
		return err
	}

	// set the permissions for the .claudemod directory
	err = ensureClaudeModPermissions(wd)
	if err != nil {
		return err
	}

	// populate the .claudemod folder
	err = populateClaudeModFolder(wd)
	if err != nil {
		return err
	}

	return nil
}

func getClaudeFolderPath(wd string) string {
	return filepath.Join(wd, ".claude")
}

func getClaudeModFolderPath(wd string) string {
	return filepath.Join(wd, ".claudemod")
}

// ensureClaudeFolderExists creates a .claude directory in the current working directory if it doesn't exist. No-op if it already exists.
func ensureClaudeFolderExists(wd string) error {
	// get the .claude folder path
	claudeFolderPath := getClaudeFolderPath(wd)

	// create the .claude folder if it doesn't exist
	if !dirExists(claudeFolderPath) {
		err := os.MkdirAll(claudeFolderPath, 0755)
		if err != nil {
			return err
		}
	}

	// create the .claude/settings.local.json file if it doesn't exist
	settingsFilePath := filepath.Join(claudeFolderPath, "settings.local.json")
	if !fileExists(settingsFilePath) {
		err := os.WriteFile(settingsFilePath, []byte("{}"), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func ensureClaudeModFolderExists(wd string) error {
	// create the .claudemod folder if it doesn't exist
	claudeModFolderPath := getClaudeModFolderPath(wd)
	if !dirExists(claudeModFolderPath) {
		err := os.MkdirAll(claudeModFolderPath, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

type ClaudeSettings struct {
	Permissions struct {
		Allow []string `json:"allow"`
	} `json:"permissions"`
}

func ensureClaudeModPermissions(wd string) error {
	// ensure .claude/settings.local.json exists for permissions
	settingsFilePath := filepath.Join(getClaudeFolderPath(wd), "settings.local.json")
	if !fileExists(settingsFilePath) {
		err := os.WriteFile(settingsFilePath, []byte("{}"), 0644)
		if err != nil {
			return err
		}
	}

	// read the .claude/settings.local.json file
	settingsData, err := os.ReadFile(settingsFilePath)
	if err != nil {
		return err
	}
	var settingsJson ClaudeSettings
	err = json.Unmarshal(settingsData, &settingsJson)
	if err != nil {
		return err
	}

	// append the permissions to the settings data
	permissions := []string{
		"Read(.claudemod/*)",
		"Write(.claudemod/*)",
		"Edit(.claudemod/*)",
	}
	for _, permission := range permissions {
		if !slices.Contains(settingsJson.Permissions.Allow, permission) {
			settingsJson.Permissions.Allow = append(settingsJson.Permissions.Allow, permission)
		}
	}

	// write the settings data to the .claude/settings.local.json file
	settingsData, err = json.MarshalIndent(settingsJson, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(settingsFilePath, settingsData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func populateClaudeModFolder(wd string) error {
	// copy the refs folder to the .claudemod/refs folder
	refsFilePath := filepath.Join(getClaudeModFolderPath(wd), "refs")
	if !dirExists(refsFilePath) {
		err := os.MkdirAll(refsFilePath, 0755)
		if err != nil {
			return err
		}
	}

	// copy the embedded refs to the .claudemod/refs folder
	entries, err := refsFS.ReadDir("refs")
	if err != nil {
		return fmt.Errorf("read embedded refs: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := refsFS.ReadFile(filepath.Join("refs", entry.Name()))
		if err != nil {
			return fmt.Errorf("read embedded ref %s: %w", entry.Name(), err)
		}
		err = os.WriteFile(filepath.Join(refsFilePath, entry.Name()), content, 0644)
		if err != nil {
			return fmt.Errorf("write ref %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true // File or directory exists
	}
	// Check if the error is specifically due to the file not existing
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	// For other errors (e.g., permission issues), it's unknown or a different problem
	return false
}

func dirExists(path string) bool {
	// check if path exists and is a directory
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir()
	}

	// if the directory does not exist, return false
	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	fmt.Printf("error checking if directory exists: %v\n", err)
	return false
}
