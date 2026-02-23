package workflow

var Bootstrap = Workflow{
	Phases: []Phase{
		{
			Name:        "bootstrap",
			Description: "Explore the codebase and generate initial project specs and documentation.",
		},
	},
}
