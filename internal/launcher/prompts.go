package launcher

import "fmt"

type InitialPromptValues struct {
	ExitSignal   string
	PhaseName    string
	ExtraPrompt  string
	RollbackHint string
}

func createInitialPrompt(values InitialPromptValues) string {
	exitSignal := values.ExitSignal
	extraPrompt := values.ExtraPrompt
	rollbackHint := values.RollbackHint
	phaseName := values.PhaseName

	prompt := fmt.Sprintf(`
You are executing the ClaudeMod '%s' phase. Read .claudemod/WORKFLOW.md and follow the instructions under '## Phases > ### %s'.

%s

PHASE TRANSITION: Track the phase criteria listed in WORKFLOW.md for this phase.
When the developer indicates they want to move on (e.g., 'next phase', 'move on', 'continue'), check all criteria.

If all criteria are met, first update .twincode/spec/SESSION_STATE.md with what was accomplished and what the next phase should focus on.
Then write the file .twincode/%s with the exact content PHASE_COMPLETE as your LAST action. The session will end automatically.

If criteria are unmet, list what remains and ask the developer if they want to address it or skip.

%s
	`, phaseName, phaseName, extraPrompt, exitSignal, rollbackHint)
	return prompt
}
