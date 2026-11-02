// Package domain contiene los conceptos de la feature imports.
package domain

import (
	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
)

// Plan es un proyecto completo, ya validado, listo para guardarse como una unidad.
// El orden de sus listas es el orden del fichero.
type Plan struct {
	Project projects.Project
	Stages  []StagePlan
}

// StagePlan es una etapa con sus tareas.
type StagePlan struct {
	Stage stages.Stage
	Tasks []tasks.Task
}
