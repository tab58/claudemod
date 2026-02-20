package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tab58/claudemod/internal/app"
)

func main() {
	// parse command line arguments: "run <workflow-name>"
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: claudemod <command>\n\ncommands:\n  init                scaffold .claudemod/ and .claude/ without launching Claude\n  run <workflow-name> run a workflow\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
			os.Exit(1)
		}
		if err := app.SetupPluginFolder(wd); err != nil {
			fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("claudemod: initialized .claudemod/ scaffold")
		return

	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: claudemod run <workflow-name>\n")
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\nusage: claudemod <command>\n\ncommands:\n  init                scaffold .claudemod/ and .claude/ without launching Claude\n  run <workflow-name> run a workflow\n", os.Args[1])
		os.Exit(1)
	}

	workflowName := os.Args[2]

	// trap signals and cancel context on exit
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// create a new app
	a, err := app.NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}
	defer a.Close()

	// run the workflow
	if err := a.RunWorkflow(ctx, workflowName); err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}
}
