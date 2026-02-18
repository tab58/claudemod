package launcher

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/segmentio/ksuid"
	"github.com/tab58/claudemod/internal/claudecode/bridge"
)

type Launcher interface {
	Close() error
	SpawnInteractiveSession(params SessionParams)
}

type launcherImpl struct {
	watcher *fsnotify.Watcher
	bridge  *bridge.Bridge
	wd      string
	wg      *sync.WaitGroup

	activeSession *bridge.Session
}

func (a *launcherImpl) Close() error {
	// wait for all sessions to complete
	a.wg.Wait()

	if a.bridge != nil {
		return a.bridge.Close()
	}
	return nil
}

type SessionParams struct {
	Prompt       string
	ExitCriteria string
}

func (a *launcherImpl) SpawnInteractiveSession(params SessionParams) {
	prompt := params.Prompt
	exitCriteria := params.ExitCriteria

	id := ksuid.New().String()
	signalFilename := fmt.Sprintf("signal_%s", id)

	// spawn the claude code instance
	promptText := fmt.Sprintf(`
	%s

EXIT CRITERIA: Exit the session when the following criteria are met: %s.

To exit the session, write the file .claudemod/%s with the exact content PHASE_COMPLETE as your LAST action.
The session will end automatically.
	`, prompt, exitCriteria, signalFilename)
	claudeArgs := []string{"--append-system-prompt", promptText}
	claudeCode, err := a.bridge.Spawn("claude", claudeArgs, bridge.Config{})
	if err != nil {
		log.Println("error:", err)
		return
	}
	a.activeSession = claudeCode

	// run the watcher events
	a.bridge.Activate(claudeCode)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for {
			select {
			case event, ok := <-a.watcher.Events:
				if !ok {
					return
				}
				// Check if the event is a file creation
				if event.Op&fsnotify.Create == fsnotify.Create {
					if event.Name == signalFilename {
						claudeCode.Suspend()
					}
				}
			case err, ok := <-a.watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
}

func New(wd string) (Launcher, error) {
	// create a new watcher
	watcher, err := setupWatcher(wd)
	if err != nil {
		return nil, err
	}

	// create a new bridge
	bridge, err := bridge.New(bridge.BridgeConfig{})
	if err != nil {
		return nil, err
	}

	return &launcherImpl{
		watcher: watcher,
		bridge:  bridge,
		wd:      wd,
		wg:      &sync.WaitGroup{},
	}, nil
}

func setupWatcher(wd string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// make a .claudemod directory in the current working directory
	claudemodDir := filepath.Join(wd, ".claudemod")
	if !dirExists(filepath.Join(wd, ".claudemod")) {
		err = os.Mkdir(claudemodDir, 0755)
		if err != nil {
			return nil, err
		}
	}
	err = watcher.Add(claudemodDir)
	if err != nil {
		return nil, err
	}

	return watcher, nil
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
