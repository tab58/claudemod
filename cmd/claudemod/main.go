package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tab58/claudemod/internal/app"
)

func main() {
	// parse command line arguments: "run <workflow-name>"
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: claudemod run <workflow-name>\n")
		os.Exit(1)
	}
	var workflowName string
	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: claudemod run <workflow-name>\n")
			os.Exit(1)
		}
		workflowName = os.Args[2]
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\nusage: claudemod run <workflow-name>\n", os.Args[1])
		os.Exit(1)
	}

	// trap signals and cancel context on exit
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// create a new app
	app, err := app.NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	// run the workflow
	err = app.RunWorkflow(ctx, workflowName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudemod: %v\n", err)
		os.Exit(1)
	}
}
