package app

type Workflow struct {
	Phases []WorkflowPhase
}

func (w *Workflow) GetFirstPhase() *WorkflowPhase {
	if len(w.Phases) == 0 {
		return nil
	}
	return &w.Phases[0]
}

func (w *Workflow) GetPhase(name string) *WorkflowPhase {
	for _, phase := range w.Phases {
		if phase.Name == name {
			return &phase
		}
	}
	return nil
}

func (w *Workflow) GetNextPhase(currentPhase string) *WorkflowPhase {
	for i, phase := range w.Phases {
		if phase.Name == currentPhase {
			if i < len(w.Phases)-1 {
				return &w.Phases[i+1]
			}
			return nil
		}
	}
	return nil
}

func (w *Workflow) IsEndPhase(phaseName string) bool {
	for _, phase := range w.Phases {
		if phase.Name == phaseName {
			return true
		}
	}
	return false
}

type WorkflowPhase struct {
	Name            string
	RollbackTargets []string
}
