package workflow

import (
	"fmt"
	"testing"
)

func TestRollbackTargetsReferenceEarlierPhases(t *testing.T) {
	workflows := []struct {
		name     string
		workflow Workflow
	}{
		{"Feature", Feature},
		{"Bugfix", Bugfix},
		{"Bootstrap", Bootstrap},
		{"Explain", Explain},
	}

	for _, wf := range workflows {
		t.Run(wf.name, func(t *testing.T) {
			phaseIndex := make(map[string]int, len(wf.workflow.Phases))
			for i, phase := range wf.workflow.Phases {
				phaseIndex[phase.Name] = i
			}

			for i, phase := range wf.workflow.Phases {
				for _, target := range phase.RollbackTargets {
					targetIdx, exists := phaseIndex[target]
					if !exists {
						t.Errorf("phase %q has rollback target %q which does not exist in workflow %q",
							phase.Name, target, wf.name)
						continue
					}
					if targetIdx >= i {
						t.Errorf("phase %q (index %d) has rollback target %q (index %d) which is not an earlier phase",
							phase.Name, i, target, targetIdx)
					}
				}
			}
		})
	}
}

func TestWorkflowPhaseNamesAreUnique(t *testing.T) {
	workflows := []struct {
		name     string
		workflow Workflow
	}{
		{"Feature", Feature},
		{"Bugfix", Bugfix},
		{"Bootstrap", Bootstrap},
		{"Explain", Explain},
	}

	for _, wf := range workflows {
		t.Run(wf.name, func(t *testing.T) {
			seen := make(map[string]bool, len(wf.workflow.Phases))
			for _, phase := range wf.workflow.Phases {
				if seen[phase.Name] {
					t.Errorf("duplicate phase name %q in workflow %q", phase.Name, wf.name)
				}
				seen[phase.Name] = true
			}
		})
	}
}

func TestGetPhase(t *testing.T) {
	tests := []struct {
		name      string
		workflow  Workflow
		phaseName string
		wantNil   bool
	}{
		{"existing phase", Feature, "tdd-red", false},
		{"first phase", Feature, "discuss-feature", false},
		{"last phase", Feature, "synthesize-specs", false},
		{"design-review exists", Feature, "design-review", false},
		{"nonexistent phase", Feature, "nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.workflow.GetPhase(tt.phaseName)
			if tt.wantNil && got != nil {
				t.Errorf("GetPhase(%q) = %v, want nil", tt.phaseName, got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("GetPhase(%q) = nil, want non-nil", tt.phaseName)
			}
			if got != nil && got.Name != tt.phaseName {
				t.Errorf("GetPhase(%q).Name = %q, want %q", tt.phaseName, got.Name, tt.phaseName)
			}
		})
	}
}

func TestGetNextPhase(t *testing.T) {
	tests := []struct {
		name         string
		workflow     Workflow
		currentPhase string
		wantName     string
		wantNil      bool
	}{
		{"first to second", Feature, "discuss-feature", "spec-plan", false},
		{"middle", Feature, "scope-plan", "tdd-red", false},
		{"tdd-refactor to design-review", Feature, "tdd-refactor", "design-review", false},
		{"design-review to code-review", Feature, "design-review", "code-review", false},
		{"last phase", Feature, "synthesize-specs", "", true},
		{"nonexistent", Feature, "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.workflow.GetNextPhase(tt.currentPhase)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetNextPhase(%q) = %v, want nil", tt.currentPhase, got)
				}
				return
			}
			if got == nil {
				t.Errorf("GetNextPhase(%q) = nil, want %q", tt.currentPhase, tt.wantName)
				return
			}
			if got.Name != tt.wantName {
				t.Errorf("GetNextPhase(%q).Name = %q, want %q", tt.currentPhase, got.Name, tt.wantName)
			}
		})
	}
}

func TestGetFirstPhase(t *testing.T) {
	tests := []struct {
		name     string
		workflow Workflow
		wantName string
		wantNil  bool
	}{
		{"feature", Feature, "discuss-feature", false},
		{"bugfix", Bugfix, "describe-bug", false},
		{"bootstrap", Bootstrap, "bootstrap", false},
		{"explain", Explain, "ask-question", false},
		{"empty", Workflow{}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.workflow.GetFirstPhase()
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetFirstPhase() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Errorf("GetFirstPhase() = nil, want %q", tt.wantName)
				return
			}
			if got.Name != tt.wantName {
				t.Errorf("GetFirstPhase().Name = %q, want %q", got.Name, tt.wantName)
			}
		})
	}
}

func TestBugfixFirstPhaseIsDescribeBug(t *testing.T) {
	first := Bugfix.GetFirstPhase()
	if first == nil {
		t.Fatal("Bugfix.GetFirstPhase() = nil, want describe-bug")
	}
	if first.Name != "describe-bug" {
		t.Errorf("Bugfix first phase = %q, want %q", first.Name, "describe-bug")
	}
}

func TestRollbackTargetsAreOrdered(t *testing.T) {
	workflows := []struct {
		name     string
		workflow Workflow
	}{
		{"Feature", Feature},
		{"Bugfix", Bugfix},
		{"Bootstrap", Bootstrap},
		{"Explain", Explain},
	}

	for _, wf := range workflows {
		t.Run(wf.name, func(t *testing.T) {
			phaseIndex := make(map[string]int, len(wf.workflow.Phases))
			for i, phase := range wf.workflow.Phases {
				phaseIndex[phase.Name] = i
			}

			for _, phase := range wf.workflow.Phases {
				for j := 1; j < len(phase.RollbackTargets); j++ {
					prevIdx := phaseIndex[phase.RollbackTargets[j-1]]
					currIdx := phaseIndex[phase.RollbackTargets[j]]
					if prevIdx >= currIdx {
						t.Errorf("phase %q rollback targets not in phase order: %q (index %d) before %q (index %d)",
							phase.Name,
							phase.RollbackTargets[j-1], prevIdx,
							phase.RollbackTargets[j], currIdx)
					}
				}
			}
		})
	}
}

func TestIsEndPhase(t *testing.T) {
	tests := []struct {
		name      string
		workflow  Workflow
		phaseName string
		want      bool
	}{
		{"last phase is end", Feature, "synthesize-specs", true},
		{"first phase is not end", Feature, "discuss-feature", false},
		{"middle phase is not end", Feature, "tdd-red", false},
		{"nonexistent phase is not end", Feature, "nonexistent", false},
		{"empty workflow", Workflow{}, "anything", false},
		{"bugfix last phase", Bugfix, "synthesize-specs", true},
		{"bootstrap single phase", Bootstrap, "bootstrap", true},
		{"explain last phase", Explain, "update-specs", true},
		{"explain first phase is not end", Explain, "ask-question", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.workflow.IsEndPhase(tt.phaseName)
			if got != tt.want {
				t.Errorf("IsEndPhase(%q) = %v, want %v", tt.phaseName, got, tt.want)
			}
		})
	}
}

func TestGetPhaseReturnsSliceElement(t *testing.T) {
	wf := Workflow{
		Phases: []Phase{
			{Name: "alpha"},
			{Name: "beta"},
		},
	}

	got := wf.GetPhase("beta")
	if got == nil {
		t.Fatal("GetPhase(\"beta\") = nil, want non-nil")
	}
	if got != &wf.Phases[1] {
		t.Error("GetPhase returned pointer to copy, not to slice element")
	}
}

func ExampleWorkflow_GetPhase() {
	phase := Feature.GetPhase("tdd-red")
	fmt.Println(phase.Name)
	// Output: tdd-red
}
