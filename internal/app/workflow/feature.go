package workflow

var Feature = Workflow{
	Phases: []Phase{
		{
			Name: "discuss-feature",
		},
		{Name: "spec-plan"},
		{
			Name:            "scope-plan",
			RollbackTargets: []string{"discuss-feature", "spec-plan"},
		},
		{
			Name:            "tdd-red",
			RollbackTargets: []string{"discuss-feature", "spec-plan", "scope-plan"},
		},
		{
			Name:            "tdd-green",
			RollbackTargets: []string{"discuss-feature", "spec-plan", "scope-plan", "tdd-red"},
		},
		{
			Name:            "code-review",
			RollbackTargets: []string{"discuss-feature", "spec-plan", "scope-plan", "tdd-red", "tdd-green"},
		},
		{Name: "synthesize-specs"},
	},
}
