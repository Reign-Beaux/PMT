// Package domain contiene los planes de la feature workflow: qué elementos mueve
// una operación que cruza features. Las reglas siguen en el domain de cada feature.
package domain

import (
	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
)

// ProjectMove es el paso de un proyecto a otro status.
type ProjectMove struct {
	Project projects.Project
	Change  projects.StatusChange
}

// StageMove es el paso de una etapa a otro status. Add trae el registro pendiente
// de una etapa que sale de su planning propio.
type StageMove struct {
	Stage  stages.Stage
	Change stages.StatusChange
	Add    *stages.Change
}

// TaskMove es el paso de una tarea a otro status.
type TaskMove struct {
	Task   tasks.Task
	Change tasks.StatusChange
}

// ReleasePlan es lo que mueve release_project.
type ReleasePlan struct {
	Project ProjectMove
	Stages  []StageMove
}

// StartPlan es lo que mueve start_stage. Project es nulo si el proyecto ya salió de stand_by.
type StartPlan struct {
	Stage   StageMove
	Tasks   []TaskMove
	Project *ProjectMove
}
