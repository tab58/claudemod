package workflow

type Phase struct {
	Name            string
	RollbackTargets []string
}

type Workflow struct {
	Phases []Phase
}

func (w *Workflow) GetFirstPhase() *Phase {
	if len(w.Phases) == 0 {
		return nil
	}
	return &w.Phases[0]
}

func (w *Workflow) GetPhase(name string) *Phase {
	for _, phase := range w.Phases {
		if phase.Name == name {
			return &phase
		}
	}
	return nil
}

func (w *Workflow) GetNextPhase(currentPhase string) *Phase {
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
