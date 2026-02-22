package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tab58/claudemod/internal/app/workflow"
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
	wf, ok := definedWorkflows[workflowName]
	if !ok {
		return fmt.Errorf("workflow %s not found", workflowName)
	}
	firstPhase := wf.GetFirstPhase()
	if firstPhase == nil {
		return fmt.Errorf("invalid workflow %s: has no defined phases", workflowName)
	}

	// determine starting phase: primary=PHASE_LOG.jsonl, fallback=SESSION_STATE.json
	currentPhase := a.resolveStartingPhase(wf, firstPhase)

	// run the workflow loop
	endWorkflow := false
	for !endWorkflow {
		if err := a.runPhase(ctx, wf, currentPhase.Name); err != nil {
			if errors.Is(err, context.Canceled) {
				// append restart entry to phase log for crash recovery
				_ = a.appendPhaseLog(PhaseLogEntry{
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Phase:     currentPhase.Name,
					Action:    "restart",
				})
				return fmt.Errorf("workflow interrupted at phase %s: %w", currentPhase.Name, err)
			}
			return fmt.Errorf("error running phase %s: %w", currentPhase.Name, err)
		}

		// read Claude's transient session state
		sessionState, err := a.readSessionState()
		if err != nil {
			return fmt.Errorf("unable to read session state: %w", err)
		}

		// promote session state into the append-only phase log
		logEntry := PhaseLogEntry{
			Timestamp:         time.Now().UTC().Format(time.RFC3339),
			Phase:             currentPhase.Name,
			Action:            sessionState.Action,
			DiscussionSummary: sessionState.DiscussionSummary,
			Recommendation:    sessionState.Recommendation,
			Explanation:       sessionState.Explanation,
		}

		switch sessionState.Action {
		case "advance":
			nextPhase := wf.GetNextPhase(currentPhase.Name)
			if nextPhase == nil {
				logEntry.Action = "complete"
				if err := a.appendPhaseLog(logEntry); err != nil {
					return fmt.Errorf("failed to append phase log: %w", err)
				}
				endWorkflow = true
			} else {
				if err := a.appendPhaseLog(logEntry); err != nil {
					return fmt.Errorf("failed to append phase log: %w", err)
				}
				currentPhase = nextPhase
			}
		case "complete":
			if err := a.appendPhaseLog(logEntry); err != nil {
				return fmt.Errorf("failed to append phase log: %w", err)
			}
			endWorkflow = true
		case "restart", "rollback":
			// validate the target phase before writing to log
			rollbackPhase := wf.GetPhase(sessionState.Phase)
			if rollbackPhase == nil {
				return fmt.Errorf("invalid session state phase: %s", sessionState.Phase)
			}
			if sessionState.Phase != "" {
				logEntry.Phase = sessionState.Phase
			}
			if err := a.appendPhaseLog(logEntry); err != nil {
				return fmt.Errorf("failed to append phase log: %w", err)
			}
			currentPhase = rollbackPhase
		default:
			return fmt.Errorf("invalid session state action: %s", sessionState.Action)
		}
	}
	return nil
}

// resolveStartingPhase determines which phase to start from by reading the
// phase log (primary) or SESSION_STATE.json (fallback for backward compat).
func (a *App) resolveStartingPhase(wf *workflow.Workflow, firstPhase *workflow.Phase) *workflow.Phase {
	// Primary: read PHASE_LOG.jsonl
	entries, err := a.readPhaseLog()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: warning: failed to read phase log, falling back to session state: %v\n", err)
	} else if len(entries) > 0 {
		last := entries[len(entries)-1]
		return a.phaseFromLogEntry(wf, last, firstPhase)
	}

	// Fallback: SESSION_STATE.json (backward compatibility)
	savedState, err := a.readSessionState()
	if err == nil {
		switch savedState.Action {
		case "restart", "rollback":
			if p := wf.GetPhase(savedState.Phase); p != nil {
				fmt.Fprintf(os.Stderr, "claudemod: resuming at phase %q (from session state)\n", p.Name)
				return p
			}
		case "advance":
			if savedState.Phase != "" {
				if p := wf.GetNextPhase(savedState.Phase); p != nil {
					fmt.Fprintf(os.Stderr, "claudemod: resuming at phase %q (from session state)\n", p.Name)
					return p
				}
			}
		case "complete":
			// workflow was finished; start fresh
		}
	}

	return firstPhase
}

// phaseFromLogEntry maps a phase log entry to the appropriate starting phase.
func (a *App) phaseFromLogEntry(wf *workflow.Workflow, entry PhaseLogEntry, firstPhase *workflow.Phase) *workflow.Phase {
	switch entry.Action {
	case "restart", "rollback":
		if p := wf.GetPhase(entry.Phase); p != nil {
			fmt.Fprintf(os.Stderr, "claudemod: resuming at phase %q\n", p.Name)
			return p
		}
		fmt.Fprintf(os.Stderr, "claudemod: warning: phase log references unknown phase %q, starting from first phase\n", entry.Phase)
	case "advance":
		if p := wf.GetNextPhase(entry.Phase); p != nil {
			fmt.Fprintf(os.Stderr, "claudemod: resuming at phase %q\n", p.Name)
			return p
		}
		fmt.Fprintf(os.Stderr, "claudemod: warning: no phase after %q, starting from first phase\n", entry.Phase)
	case "complete":
		// workflow was finished; start fresh
	}
	return firstPhase
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

func (a *App) writeSessionState(state SessionState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session state: %w", err)
	}
	path := filepath.Join(a.claudeModDir, "SESSION_STATE.json")
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write session state tmp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename session state: %w", err)
	}
	return nil
}

func (a *App) runPhase(ctx context.Context, wf *workflow.Workflow, phaseName string) error {
	wfValues := buildWorkflowValues()

	// read phase log for history context
	entries, err := a.readPhaseLog()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: warning: failed to read phase log: %v\n", err)
	}
	phaseLogContent := formatPhaseLog(entries)

	// get rollback targets for the current phase
	var rollbackTargets []string
	if phase := wf.GetPhase(phaseName); phase != nil {
		rollbackTargets = phase.RollbackTargets
	}

	// render phase instructions from the phase template
	renderedInstructions, err := renderPhaseInstructions(phaseName, wfValues)
	if err != nil {
		return fmt.Errorf("render phase instructions for %s: %w", phaseName, err)
	}

	// build full system prompt values
	values := SystemPromptValues{
		WorkflowValues:  wfValues,
		PhaseName:       phaseName,
		RollbackTargets: rollbackTargets,
		ExtraPrompt:     "",
		PhaseLogFileName: phaseLogFileName,

		RenderedPhaseInstructions: renderedInstructions,
		PhaseLogContent:           phaseLogContent,
	}

	systemPrompt, err := generateSystemPrompt(values)
	if err != nil {
		return fmt.Errorf("generate system prompt for %s: %w", phaseName, err)
	}

	agentPrompt := "Begin."
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
