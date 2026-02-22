package workflow

var Bugfix = Workflow{
	Phases: []Phase{
		{
			Name: "describe-bug",
		},
		{
			Name:            "spec-plan",
			RollbackTargets: []string{"describe-bug"},
		},
		{
			Name:            "scope-plan",
			RollbackTargets: []string{"describe-bug", "spec-plan"},
		},
		{
			Name:            "tdd-red",
			RollbackTargets: []string{"describe-bug", "spec-plan", "scope-plan"},
		},
		{
			Name:            "tdd-green",
			RollbackTargets: []string{"describe-bug", "spec-plan", "scope-plan", "tdd-red"},
		},
		{
			Name:            "code-review",
			RollbackTargets: []string{"describe-bug", "spec-plan", "scope-plan", "tdd-red", "tdd-green"},
		},
		{Name: "synthesize-specs"},
	},
}
