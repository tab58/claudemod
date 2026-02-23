package workflow

var Explain = Workflow{
	Phases: []Phase{
		{
			Name:        "ask-question",
			Description: "Gather the developer's question and identify which parts of the codebase to investigate.",
		},
		{
			Name:            "deep-dive",
			Description:     "Perform a thorough investigation of the relevant code to build a detailed answer.",
			RollbackTargets: []string{"ask-question"},
		},
		{
			Name:            "update-specs",
			Description:     "Update project specs and documentation with any new understanding gained from the investigation.",
			RollbackTargets: []string{"ask-question", "deep-dive"},
		},
	},
}
