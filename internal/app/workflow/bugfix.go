package workflow

var Bugfix = Workflow{
	Phases: []Phase{
		{
			Name:        "describe-bug",
			Description: "Describe the bug with the developer, gathering reproduction steps, expected vs. actual behavior, and impact.",
		},
		{
			Name:            "spec-plan",
			Description:     "Draft a technical specification for the fix, documenting root cause analysis and the agreed-upon approach.",
			RollbackTargets: []string{"describe-bug"},
		},
		{
			Name:            "scope-plan",
			Description:     "Break the fix into an ordered implementation plan with concrete, incremental steps.",
			RollbackTargets: []string{"describe-bug", "spec-plan"},
		},
		{
			Name:            "tdd-red",
			Description:     "Write failing tests that reproduce the bug and encode the expected correct behavior.",
			RollbackTargets: []string{"describe-bug", "spec-plan", "scope-plan"},
		},
		{
			Name:            "tdd-green",
			Description:     "Write the minimal fix needed to make all failing tests pass.",
			RollbackTargets: []string{"describe-bug", "spec-plan", "scope-plan", "tdd-red"},
		},
		{
			Name:            "tdd-refactor",
			Description:     "Refactor the fix for clarity and maintainability while keeping all tests green.",
			RollbackTargets: []string{"describe-bug", "spec-plan", "scope-plan", "tdd-red", "tdd-green"},
		},
		{
			Name:            "design-review",
			Description:     "Review the fix for architectural quality, regression safety, and adherence to the spec.",
			RollbackTargets: []string{"describe-bug", "spec-plan", "scope-plan", "tdd-red", "tdd-green", "tdd-refactor"},
		},
		{
			Name:            "code-review",
			Description:     "Perform a final code review for correctness, security, style, and readiness to merge.",
			RollbackTargets: []string{"describe-bug", "spec-plan", "scope-plan", "tdd-red", "tdd-green", "tdd-refactor", "design-review"},
		},
		{
			Name:        "synthesize-specs",
			Description: "Update project specs and documentation to reflect the completed fix.",
		},
	},
}
