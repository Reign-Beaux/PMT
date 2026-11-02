// Package progress consulta el avance de un proyecto: sus etapas, sus tareas, sus
// bugs abiertos y las señales de un posible atasco.
package progress

import (
	"context"
	"time"

	"github.com/google/uuid"

	bugs "pmt/internal/features/bugs/domain"
	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
)

// Códigos de las señales.
const (
	SignalReadyWithoutWork = "ready_without_work"
	SignalStageNotClosed   = "stage_not_closed"
	SignalStageNotStarted  = "stage_not_started"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Project(ctx context.Context, id uuid.UUID) (projects.Project, error)
	Stages(ctx context.Context, projectID uuid.UUID) ([]stages.Stage, error)
	Tasks(ctx context.Context, stageID uuid.UUID) ([]tasks.Task, error)
	Bugs(ctx context.Context, projectID uuid.UUID) ([]bugs.Bug, error)
}

// Project es el proyecto. LastChangeAt es la última actualización del proyecto, de
// sus etapas o de sus tareas sin retirar; los bugs no la mueven.
type Project struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Status       projects.Status `json:"status"`
	LastChangeAt time.Time       `json:"last_change_at"`
}

// Task es una tarea de la etapa.
type Task struct {
	ID        uuid.UUID    `json:"id"`
	Title     string       `json:"title"`
	Status    tasks.Status `json:"status"`
	Assignee  string       `json:"assignee,omitempty"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// Stage es una etapa con sus tareas. Counts solo incluye los status presentes.
type Stage struct {
	ID      uuid.UUID      `json:"id"`
	Name    string         `json:"name"`
	Status  stages.Status  `json:"status"`
	Lead    string         `json:"lead,omitempty"`
	Members []string       `json:"members"`
	Counts  map[string]int `json:"counts"`
	Tasks   []Task         `json:"tasks"`
}

// Bug es un bug abierto del proyecto.
type Bug struct {
	ID        uuid.UUID   `json:"id"`
	Title     string      `json:"title"`
	Status    bugs.Status `json:"status"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// OpenBugs son los bugs que no están en done ni rejected, por fecha de creación.
// Counts solo incluye los status presentes.
type OpenBugs struct {
	Counts map[string]int `json:"counts"`
	Items  []Bug          `json:"items"`
}

// Signal es un hecho que apunta a un posible atasco, sobre una etapa.
type Signal struct {
	Code      string    `json:"code"`
	StageID   uuid.UUID `json:"stage_id"`
	StageName string    `json:"stage_name"`
}

// Result es el avance del proyecto.
type Result struct {
	Project  Project  `json:"project"`
	Stages   []Stage  `json:"stages"`
	OpenBugs OpenBugs `json:"open_bugs"`
	Signals  []Signal `json:"signals"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve el avance. Lo retirado no aparece.
func (h *Handler) Handle(ctx context.Context, projectID uuid.UUID) (Result, error) {
	project, err := h.reader.Project(ctx, projectID)
	if err != nil {
		return Result{}, err
	}
	stageList, err := h.reader.Stages(ctx, projectID)
	if err != nil {
		return Result{}, err
	}
	bugList, err := h.reader.Bugs(ctx, projectID)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Project:  Project{ID: project.ID, Name: project.Name, Status: project.Status, LastChangeAt: project.UpdatedAt},
		Stages:   make([]Stage, 0, len(stageList)),
		OpenBugs: openBugs(bugList),
		Signals:  []Signal{},
	}
	for _, s := range stageList {
		stage := Stage{
			ID: s.ID, Name: s.Name, Status: s.Status, Lead: s.Lead,
			Members: append([]string{}, s.Members...),
			Counts:  map[string]int{},
			Tasks:   []Task{},
		}
		result.Project.LastChangeAt = latest(result.Project.LastChangeAt, s.UpdatedAt)

		taskList, err := h.reader.Tasks(ctx, s.ID)
		if err != nil {
			return Result{}, err
		}
		for _, t := range taskList {
			stage.Tasks = append(stage.Tasks, Task{ID: t.ID, Title: t.Title, Status: t.Status, Assignee: t.Assignee, UpdatedAt: t.UpdatedAt})
			stage.Counts[string(t.Status)]++
			result.Project.LastChangeAt = latest(result.Project.LastChangeAt, t.UpdatedAt)
		}

		result.Stages = append(result.Stages, stage)
		if code := stageSignal(project.Status, stage); code != "" {
			result.Signals = append(result.Signals, Signal{Code: code, StageID: s.ID, StageName: s.Name})
		}
	}
	if next := nextStage(project.Status, result.Stages); next != nil {
		result.Signals = append(result.Signals, Signal{Code: SignalStageNotStarted, StageID: next.ID, StageName: next.Name})
	}
	return result, nil
}

func openBugs(list []bugs.Bug) OpenBugs {
	open := OpenBugs{Counts: map[string]int{}, Items: []Bug{}}
	for _, b := range list {
		if !b.Status.IsOpen() {
			continue
		}
		open.Items = append(open.Items, Bug{ID: b.ID, Title: b.Title, Status: b.Status, UpdatedAt: b.UpdatedAt})
		open.Counts[string(b.Status)]++
	}
	return open
}

// stageSignal devuelve la señal de una etapa en in_dev, si la tiene. Con el proyecto
// en stopped o cancelled no hay: nadie debe seguir esa etapa.
func stageSignal(project projects.Status, stage Stage) string {
	if project == projects.StatusStopped || project == projects.StatusCancelled {
		return ""
	}
	if stage.Status != stages.StatusInDev {
		return ""
	}
	count := func(status tasks.Status) int { return stage.Counts[string(status)] }

	inFlight := count(tasks.StatusInDev) + count(tasks.StatusInReview) + count(tasks.StatusChangesRequested) +
		count(tasks.StatusInQA) + count(tasks.StatusPendingClose)
	if count(tasks.StatusReady) > 0 && inFlight == 0 {
		return SignalReadyWithoutWork
	}
	done := count(tasks.StatusDone)
	if done > 0 && done+count(tasks.StatusCancelled) == len(stage.Tasks) {
		return SignalStageNotClosed
	}
	return ""
}

// nextStage devuelve la primera etapa en stand_by si el proyecto acepta iniciarla y
// ninguna está en in_dev.
func nextStage(status projects.Status, list []Stage) *Stage {
	switch status {
	case projects.StatusStandBy, projects.StatusInDev, projects.StatusWorking:
	default:
		return nil
	}
	var next *Stage
	for i := range list {
		switch list[i].Status {
		case stages.StatusInDev:
			return nil
		case stages.StatusStandBy:
			if next == nil {
				next = &list[i]
			}
		}
	}
	return next
}

func latest(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
