package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/segmentio/ksuid"
	"github.com/tab58/claudemod/internal/claudecode/bridge"
)

type Launcher interface {
	Close() error
	SpawnInteractiveSession(ctx context.Context, params SessionParams) error
}

type launcherImpl struct {
	watcher       *fsnotify.Watcher
	bridge        *bridge.Bridge
	wd            string
	claudeModDir  string
	activeSession *bridge.Session
}

func (a *launcherImpl) Close() error {
	if a.bridge != nil {
		return a.bridge.Close()
	}
	if a.watcher != nil {
		return a.watcher.Close()
	}
	return nil
}

type SessionParams struct {
	Prompt       string
	ExitCriteria string
	WorkingDir   string
}

func (a *launcherImpl) SpawnInteractiveSession(ctx context.Context, params SessionParams) error {
	prompt := params.Prompt
	exitCriteria := params.ExitCriteria

	id := ksuid.New().String()
	signalRelativePath := ".claude/claudemod/" + fmt.Sprintf("signal_%s", id)
	signalFilepath := filepath.Join(a.wd, signalRelativePath)

	// spawn the claude code instance
	wd := params.WorkingDir
	if wd == "" {
		wd = a.wd
	}
	promptText := fmt.Sprintf(`
	%s

EXIT CRITERIA: Exit the session when the following criteria are met: %s.

To exit the session, write the file %s with the exact content {"exit_criteria_met": true} as your LAST action.
The session will end automatically.
	`, prompt, exitCriteria, signalRelativePath)
	claudeArgs := []string{
		"--append-system-prompt", promptText,
		"--add-dir", wd,
		"--allowedTools", strings.Join(
			[]string{
				"Read(.claude/claudemod/**)",
				"Write(.claude/claudemod/**)",
				"Edit(.claude/claudemod/**)",
			},
			",",
		),
	}
	fmt.Println("prompt:", promptText)
	claudeCode, err := a.bridge.Spawn("claude", claudeArgs, bridge.Config{})
	if err != nil {
		return fmt.Errorf("spawn session: %w", err)
	}
	a.activeSession = claudeCode
	a.bridge.Activate(claudeCode)

	// watch for signal file creation
	exitSignalDetected := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case event, ok := <-a.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create && event.Name == signalFilepath {
					select {
					case exitSignalDetected <- struct{}{}:
					default:
					}
					return
				}
			case watchErr, ok := <-a.watcher.Errors:
				if !ok {
					return
				}
				log.Println("watcher error:", watchErr)
			}
		}
	}()

	// block until context cancelled, signal file appears, or child exits
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-exitSignalDetected:
		err := claudeCode.Suspend()
		if err != nil {
			return fmt.Errorf("error suspending session: %w", err)
		}
		a.activeSession = nil
		err = os.Remove(signalFilepath)
		if err != nil {
			return fmt.Errorf("error removing signal file: %w", err)
		}
		return nil
	case <-claudeCode.Done():
		exitCode, exitErr := claudeCode.Wait()
		if exitErr != nil {
			return exitErr
		}
		if exitCode != 0 {
			return fmt.Errorf("session exited with code %d", exitCode)
		}
		return nil
	}
}

func New(wd string) (Launcher, error) {
	// create a new watcher
	claudeModDir, watcher, err := setupWatcher(wd)
	if err != nil {
		return nil, err
	}

	// create a new bridge
	bridge, err := bridge.New(bridge.BridgeConfig{})
	if err != nil {
		return nil, err
	}

	return &launcherImpl{
		watcher:      watcher,
		bridge:       bridge,
		wd:           wd,
		claudeModDir: claudeModDir,
	}, nil
}

func setupWatcher(wd string) (string, *fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return "", nil, err
	}

	// make a .claude directory in the current working directory
	claudeDir := filepath.Join(wd, ".claude")
	if !dirExists(claudeDir) {
		// create the .claude directory
		err = os.MkdirAll(claudeDir, 0755)
		if err != nil {
			return "", nil, err
		}
	}

	// make a .claude/claudemod directory in the current working directory
	claudeModDir := filepath.Join(claudeDir, "claudemod")
	if !dirExists(claudeModDir) {
		// create the .claude/claudemod directory
		err = os.MkdirAll(claudeModDir, 0755)
		if err != nil {
			return "", nil, err
		}
	}
	err = watcher.Add(claudeModDir)
	if err != nil {
		return "", nil, err
	}

	// setup the .claude directory (disabled for now)
	// err = setupClaudeFolder(claudeDir)
	// if err != nil {
	// 	return "", nil, err
	// }

	return claudeModDir, watcher, nil
}

type ClaudeSettingsJSON struct {
	Permissions struct {
		Allow []string `json:"allow"`
	} `json:"permissions"`
}

func setupClaudeFolder(claudeDir string) error {
	fmt.Println("setting up .claude directory...")

	// create the .claude/settings.local.json file if it doesn't exist
	if !fileExists(filepath.Join(claudeDir, "settings.local.json")) {
		err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte("{}"), 0644)
		if err != nil {
			return err
		}
	}
	// read the .claude/settings.local.json file
	settingsFilePath := filepath.Join(claudeDir, "settings.local.json")
	data, err := os.ReadFile(settingsFilePath)
	if err != nil {
		return err
	}
	// append the permissions to the settings data
	permissions := []string{
		"Read(.claude/claudemod/*)",
		"Write(.claude/claudemod/*)",
		"Edit(.claude/claudemod/*)",
	}
	data, err = appendPermissions(data, permissions)
	if err != nil {
		return err
	}
	// write the settings data to the .claude/settings.local.json file
	err = os.WriteFile(settingsFilePath, data, 0644)
	if err != nil {
		return err
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

func appendPermissions(settingsData []byte, permissions []string) ([]byte, error) {
	// parse the .claude/settings.local.json file
	var settingsJSON ClaudeSettingsJSON
	err := json.Unmarshal(settingsData, &settingsJSON)
	if err != nil {
		return nil, err
	}
	// add the permission allow array to the .claude/settings.local.json file
	for _, permission := range permissions {
		if !slices.Contains(settingsJSON.Permissions.Allow, permission) {
			settingsJSON.Permissions.Allow = append(settingsJSON.Permissions.Allow, permission)
		}
	}
	// marshal the settingsJSON to a JSON string
	return json.MarshalIndent(settingsJSON, "", "  ")
}
