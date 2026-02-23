package workflow

var Feature = Workflow{
	Phases: []Phase{
		{
			Name:        "discuss-feature",
			Description: "Discuss the feature with the developer to clarify requirements, constraints, and scope.",
		},
		{
			Name:            "spec-plan",
			Description:     "Draft a technical specification based on the discussion, documenting the agreed-upon design and acceptance criteria.",
			RollbackTargets: []string{"discuss-feature"},
		},
		{
			Name:            "scope-plan",
			Description:     "Break the spec into an ordered implementation plan with concrete, incremental steps.",
			RollbackTargets: []string{"discuss-feature", "spec-plan"},
		},
		{
			Name:            "tdd-red",
			Description:     "Write failing tests that encode the expected behavior from the spec before any implementation.",
			RollbackTargets: []string{"discuss-feature", "spec-plan", "scope-plan"},
		},
		{
			Name:            "tdd-green",
			Description:     "Write the minimal implementation code needed to make all failing tests pass.",
			RollbackTargets: []string{"discuss-feature", "spec-plan", "scope-plan", "tdd-red"},
		},
		{
			Name:            "tdd-refactor",
			Description:     "Refactor the implementation for clarity and maintainability while keeping all tests green.",
			RollbackTargets: []string{"discuss-feature", "spec-plan", "scope-plan", "tdd-red", "tdd-green"},
		},
		{
			Name:            "design-review",
			Description:     "Review the implementation for architectural quality, design patterns, and adherence to the spec.",
			RollbackTargets: []string{"discuss-feature", "spec-plan", "scope-plan", "tdd-red", "tdd-green", "tdd-refactor"},
		},
		{
			Name:            "code-review",
			Description:     "Perform a final code review for correctness, security, style, and readiness to merge.",
			RollbackTargets: []string{"discuss-feature", "spec-plan", "scope-plan", "tdd-red", "tdd-green", "tdd-refactor", "design-review"},
		},
		{
			Name:        "synthesize-specs",
			Description: "Update project specs and documentation to reflect the completed feature.",
		},
	},
}
