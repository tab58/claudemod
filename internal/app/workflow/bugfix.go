package workflow

var Bugfix = Workflow{
	Phases: []Phase{
		{
			Name:            "describe-bug",
			RollbackTargets: []string{"bootstrap"},
		},
		{Name: "spec-plan"},
		{
			Name:            "scope-plan",
			RollbackTargets: []string{"describe-bug", "spec-plan"},
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
