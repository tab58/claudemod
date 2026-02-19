package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/tab58/claudemod/internal/launcher"
)

type App struct {
	wd           string
	lnch         launcher.Launcher
	claudeModDir string
	watcher      *fsnotify.Watcher
}

func NewApp() (*App, error) {
	// get the working directory
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}

	// create a new launcher
	lnch, err := launcher.New(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}

	// setup the ClaudeMod folder
	err = SetupPluginFolder(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}

	return &App{
		wd:           wd,
		lnch:         lnch,
		claudeModDir: getClaudeModFolderPath(wd),
	}, nil
}

func (a *App) Close() error {
	return a.lnch.Close()
}

type SessionState struct {
	Action            string `json:"action"`
	Phase             string `json:"phase"`
	DiscussionSummary string `json:"discussion_summary"`
	Recommendation    string `json:"recommendation"`
	Explanation       string `json:"explanation"`
}

func (a *App) RunWorkflow(ctx context.Context, workflowName string) error {
	workflow, ok := definedWorkflows[workflowName]
	if !ok {
		return fmt.Errorf("workflow %s not found", workflowName)

	}
	firstPhase := workflow.GetFirstPhase()
	if firstPhase == nil {
		return fmt.Errorf("invalid workflow %s: has no defined phases", workflowName)
	}

	// run the workflow loop
	currentPhase := firstPhase
	endWorkflow := false
	for !endWorkflow {
		// run the current phase
		if err := a.runPhase(ctx, currentPhase.Name); err != nil {
			return fmt.Errorf("error running phase %s: %w", currentPhase.Name, err)
		}

		// read the session state
		sessionState, err := a.readSessionState()
		if err != nil {
			return fmt.Errorf("unable to read session state: %w", err)
		}
		switch sessionState.Action {
		case "advance":
			nextPhase := workflow.GetNextPhase(currentPhase.Name)
			if nextPhase == nil {
				endWorkflow = true
			} else {
				currentPhase = nextPhase
			}
		case "complete":
			endWorkflow = true
		case "restart":
			fallthrough
		case "rollback":
			rollbackPhase := workflow.GetPhase(sessionState.Phase)
			if rollbackPhase == nil {
				return fmt.Errorf("invalid session state phase: %s", sessionState.Phase)
			}
			currentPhase = workflow.GetPhase(sessionState.Phase)
		default:
			return fmt.Errorf("invalid session state action: %s", sessionState.Action)
		}
	}
	return nil
}

func (a *App) readSessionState() (*SessionState, error) {
	sessionStateData, err := os.ReadFile(filepath.Join(a.claudeModDir, "SESSION_STATE.json"))
	if err != nil {
		return nil, fmt.Errorf("read session state: %w", err)
	}
	var sessionState SessionState
	if err := json.Unmarshal(sessionStateData, &sessionState); err != nil {
		return nil, fmt.Errorf("unmarshal session state: %w", err)
	}
	return &sessionState, nil
}

func (a *App) runPhase(ctx context.Context, phaseName string) error {
	// generate the system and agent prompts
	systemPrompt, err := generateSystemPrompt(SystemPromptValues{
		PhaseName:   phaseName,
		ExtraPrompt: "",
	})
	if err != nil {
		return err
	}
	agentPrompt := "Begin."

	// run the session
	return a.lnch.SpawnInteractiveSession(
		ctx,
		launcher.WithSystemPrompt(systemPrompt),
		launcher.WithAgentPrompt(agentPrompt),
		launcher.WithPermissions([]string{
			fmt.Sprintf("Read(%s/*)", a.claudeModDir),
			fmt.Sprintf("Write(%s/*)", a.claudeModDir),
			fmt.Sprintf("Edit(%s/*)", a.claudeModDir),
		}),
	)
}
