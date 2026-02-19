package launcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/segmentio/ksuid"
	"github.com/tab58/claudemod/internal/claudecode/bridge"
)

type Launcher interface {
	Close() error
	SpawnInteractiveSession(ctx context.Context, options ...SessionOption) error
}

type launcherImpl struct {
	watcher       *fsnotify.Watcher
	bridge        *bridge.Bridge
	wd            string
	activeSession *bridge.Session
}

func (a *launcherImpl) Close() error {
	if a.activeSession != nil {
		return a.activeSession.Suspend()
	}
	if a.bridge != nil {
		return a.bridge.Close()
	}
	if a.watcher != nil {
		return a.watcher.Close()
	}
	return nil
}

type sessionOptions struct {
	systemPrompt string
	agentPrompt  string
	permissions  []string
}

type SessionOption func(*sessionOptions)

func WithSystemPrompt(systemPrompt string) SessionOption {
	return func(o *sessionOptions) {
		o.systemPrompt = systemPrompt
	}
}

func WithAgentPrompt(agentPrompt string) SessionOption {
	return func(o *sessionOptions) {
		o.agentPrompt = agentPrompt
	}
}

func WithPermissions(permissions []string) SessionOption {
	return func(o *sessionOptions) {
		o.permissions = permissions
	}
}

func (a *launcherImpl) SpawnInteractiveSession(ctx context.Context, options ...SessionOption) error {
	// default options
	opts := sessionOptions{
		permissions: make([]string, 0),
	}
	for _, option := range options {
		option(&opts)
	}

	// get the signal file path
	id := ksuid.New().String()
	signalFilePath := filepath.Join(a.wd, fmt.Sprintf("signal_%s", id))

	agentPrompt := opts.agentPrompt
	systemPrompt := generateSystemPrompt(opts.systemPrompt, id)

	// build the claude code arguments
	// NOTE: --add-dir is variadic (<directories...>) so it will consume
	// all subsequent non-flag arguments. Use "--" to terminate flag
	// parsing before the positional prompt argument.
	claudeArgs := make([]string, 0)
	if systemPrompt != "" {
		claudeArgs = append(claudeArgs, "--append-system-prompt", systemPrompt)
	}
	if a.wd != "" {
		claudeArgs = append(claudeArgs, "--add-dir", a.wd)
	}
	if agentPrompt != "" {
		claudeArgs = append(claudeArgs, "--", agentPrompt)
	}

	// spawn the claude code instance
	claudeExecPath, err := resolveClaudePath()
	if err != nil {
		return fmt.Errorf("resolve claude path: %w", err)
	}
	claudeCode, err := a.bridge.Spawn(claudeExecPath, claudeArgs, bridge.Config{})
	if err != nil {
		return fmt.Errorf("spawn session: %w", err)
	}

	// activate the session
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
				if event.Op&fsnotify.Create == fsnotify.Create && event.Name == signalFilePath {
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
		err = os.Remove(signalFilePath)
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
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(wd)
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
	}, nil
}

func generateSystemPrompt(mainPrompt string, id string) string {
	fullPromptTemplate := `%s

To exit this session, write the file "signal_%s" with the exact content {"exit_criteria_met": true} as your LAST action.
The session will end automatically.
	`
	return fmt.Sprintf(fullPromptTemplate, mainPrompt, id)
}
