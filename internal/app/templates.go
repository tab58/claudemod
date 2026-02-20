package app

import (
	"bytes"
	_ "embed"

	"github.com/tab58/claudemod/internal/utils"
)

//go:embed workflow.go.tmpl
var workflowTemplateText string
var workflowTemplate = utils.TemplateMust(utils.NewTemplate("workflow").Parse(workflowTemplateText))

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

func generateWorkflowFile(values WorkflowValues) (string, error) {
	var buf bytes.Buffer
	if err := workflowTemplate.Execute(&buf, values); err != nil {
		return "", err
	}
	return buf.String(), nil
}

//go:embed prompt_system.go.tmpl
var systemPromptTemplateText string
var systemPromptTemplate = utils.TemplateMust(utils.NewTemplate("system_prompt").Parse(systemPromptTemplateText))

type SystemPromptValues struct {
	PhaseName       string
	ExtraPrompt     string
	RollbackTargets []string
}

func generateSystemPrompt(values SystemPromptValues) (string, error) {
	var buf bytes.Buffer
	if err := systemPromptTemplate.Execute(&buf, values); err != nil {
		return "", err
	}
	return buf.String(), nil
}
