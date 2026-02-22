package workflow

var Explain = Workflow{
	Phases: []Phase{
		{Name: "ask-question"},
		{
			Name:            "deep-dive",
			RollbackTargets: []string{"ask-question"},
		},
		{
			Name:            "update-specs",
			RollbackTargets: []string{"ask-question", "deep-dive"},
		},
	},
}
