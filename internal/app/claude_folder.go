package app

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

//go:embed refs/SPEC.md
var refSpecMDString string

//go:embed refs/SPEC_INDEX.md
var refSpecIndexMDString string

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

	// set the permissions for the .claude/claudemod directory
	err = ensureClaudeModPermissions(wd)
	if err != nil {
		return err
	}

	// populate the .claude/claudemod folder
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
	return filepath.Join(getClaudeFolderPath(wd), "claudemod")
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
	// create the .claude/claudemod folder if it doesn't exist
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
	// create the .claude/claudemod/settings.local.json file if it doesn't exist
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
		"Read(.claude/claudemod/*)",
		"Write(.claude/claudemod/*)",
		"Edit(.claude/claudemod/*)",
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
	// copy the refs folder to the .claude/claudemod/refs folder
	refsFilePath := filepath.Join(getClaudeModFolderPath(wd), "refs")
	if !dirExists(refsFilePath) {
		err := os.MkdirAll(refsFilePath, 0755)
		if err != nil {
			return err
		}
	}
	err := os.WriteFile(filepath.Join(refsFilePath, "SPEC.md"), []byte(refSpecMDString), 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(refsFilePath, "SPEC_INDEX.md"), []byte(refSpecIndexMDString), 0644)
	if err != nil {
		return err
	}

	// write the workflow file to the .claude/claudemod/workflow.go file
	values := WorkflowValues{
		BaseFolderPath: ".claude/claudemod",
		Spec: WorkflowSpecValues{
			FolderRelPath: "spec",
			IndexName:     "INDEX.md",
		},
		Refs: WorkflowRefsValues{
			FolderRelPath:       "refs",
			IndexExampleRefName: "SPEC_INDEX.md",
			ExampleRefName:      "SPEC.md",
		},
		SessionStateFileName: "SESSION_STATE.md",
		TaskFileName:         "FIX_PLAN.md",
		ChangelogFileName:    "CHANGELOG.md",
	}
	workflowFile, err := generateWorkflowFile(values)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(getClaudeModFolderPath(wd), "WORKFLOW.md"), []byte(workflowFile), 0644)
	if err != nil {
		return err
	}
	return nil
}
