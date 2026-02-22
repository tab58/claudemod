package app

import (
	"bytes"
	"embed"
	"io/fs"
	"text/template"

	"github.com/tab58/claudemod/internal/app/workflow"
	"github.com/tab58/claudemod/internal/utils"
)

//go:embed templates/*.go.tmpl templates/phases/*.go.tmpl
var templatesFS embed.FS

var definedWorkflows = map[string]*workflow.Workflow{
	"bootstrap": &workflow.Bootstrap,
	"feature":   &workflow.Feature,
	"bugfix":    &workflow.Bugfix,
}

type WorkflowValues struct {
	BaseFolderPath       string
	Spec                 WorkflowSpecValues
	Refs                 WorkflowRefsValues
	SessionStateFileName string
	TaskFileName         string
	ChangelogFileName    string
	PlanFileName         string
}

type WorkflowSpecValues struct {
	FolderRelPath string
	IndexName     string
}

type WorkflowRefsValues struct {
	FolderRelPath       string
	IndexExampleRefName string
	ExampleRefName      string
	BugExampleRefName   string
}

// SystemPromptValues carries everything the system prompt template needs.
type SystemPromptValues struct {
	WorkflowValues
	PhaseName       string
	RollbackTargets []string
	ExtraPrompt     string

	PhaseLogFileName string

	// Pre-rendered content injected by Go code
	RenderedPhaseInstructions string
	PhaseLogContent           string
}

var phaseTemplateSet *template.Template
var systemPromptTemplate *utils.Template

func init() {
	// Parse phase templates into their own template set
	tmpl, err := template.New("phases").Funcs(utils.Funcs).ParseFS(templatesFS, "templates/phases/*.go.tmpl")
	if err != nil {
		panic("parse phase templates: " + err.Error())
	}
	phaseTemplateSet = tmpl

	// Parse system prompt template separately
	content, err := fs.ReadFile(templatesFS, "templates/prompt_system.go.tmpl")
	if err != nil {
		panic("read system prompt template: " + err.Error())
	}
	spt, err := utils.NewTemplate("system_prompt").Parse(string(content))
	if err != nil {
		panic("parse system prompt template: " + err.Error())
	}
	systemPromptTemplate = spt
}

// renderPhaseInstructions executes a single phase template with WorkflowValues
// and returns the rendered string.
func renderPhaseInstructions(phaseName string, values WorkflowValues) (string, error) {
	var buf bytes.Buffer
	if err := phaseTemplateSet.ExecuteTemplate(&buf, phaseName+".go.tmpl", values); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// buildWorkflowValues returns the hardcoded path conventions used across all workflows.
func buildWorkflowValues() WorkflowValues {
	return WorkflowValues{
		BaseFolderPath: ".claudemod",
		Spec: WorkflowSpecValues{
			FolderRelPath: "spec",
			IndexName:     "INDEX.md",
		},
		Refs: WorkflowRefsValues{
			FolderRelPath:       "refs",
			IndexExampleRefName: "SPEC_INDEX.md",
			ExampleRefName:      "SPEC.md",
			BugExampleRefName:   "BUG_SPEC.md",
		},
		SessionStateFileName: "SESSION_STATE.json",
		TaskFileName:         "FIX_PLAN.md",
		ChangelogFileName:    "CHANGELOG.md",
		PlanFileName:         "PLAN.md",
	}
}

func generateSystemPrompt(values SystemPromptValues) (string, error) {
	var buf bytes.Buffer
	if err := systemPromptTemplate.Execute(&buf, values); err != nil {
		return "", err
	}
	return buf.String(), nil
}
