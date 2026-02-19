package app

import (
	"context"
	"fmt"
	"os"

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
	claudeModDir := getClaudeModFolderPath(wd)
	err = SetupPluginFolder(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}

	return &App{
		wd:           wd,
		lnch:         lnch,
		claudeModDir: claudeModDir,
	}, nil
}

func (a *App) Close() error {
	return a.lnch.Close()
}

func (a *App) RunFeatureWorkflow(ctx context.Context) error {
	return a.runPhase(ctx, "discuss-feature")
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

	agentPromptText := `
Read WORKFLOW.md. Determine if the bootstrap phase has been completed.
If it has not, complete it now and then move on to the %s phase.
If it already has, begin the %s phase now.
`
	agentPrompt := fmt.Sprintf(agentPromptText, phaseName, phaseName)

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
