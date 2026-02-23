package workflow

var Backlog = Workflow{
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
			Name:            "generate-stories",
			Description:     "Transform tasks into user stories with dependencies, story points, and acceptance criteria.",
			RollbackTargets: []string{"discuss-feature", "spec-plan", "scope-plan"},
		},
	},
}
