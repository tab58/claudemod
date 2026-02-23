package workflow

var Task = Workflow{
	Phases: []Phase{
		{
			Name:        "discuss-task",
			Description: "Discuss the task with the developer to clarify what needs to change and why.",
		},
		{
			Name:            "task-plan",
			Description:     "Break the task into concrete, incremental steps in a prioritized plan.",
			RollbackTargets: []string{"discuss-task"},
		},
		{
			Name:            "execute-task",
			Description:     "Execute each planned task one at a time, verifying no regressions after each.",
			RollbackTargets: []string{"discuss-task", "task-plan"},
		},
		{
			Name:            "code-review",
			Description:     "Perform a final code review for correctness, security, style, and readiness to merge.",
			RollbackTargets: []string{"discuss-task", "task-plan", "execute-task"},
		},
		{
			Name:        "synthesize-specs",
			Description: "Update project specs and documentation to reflect the completed work.",
		},
	},
}
