// Package domain contiene los conceptos y reglas de la feature devplans.
package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrPlanNotFound indica que el plan no existe.
	ErrPlanNotFound = kernel.NotFound("plan de desarrollo no encontrado")
	// ErrModeInvalid indica que el plan no pertenece a exactamente una etapa o un bug.
	ErrModeInvalid = kernel.Invalid("indica stage_id con task_ids, o bug_id: exactamente uno de los dos")
	// ErrTasksRequired indica que un plan de etapa no vincula ninguna tarea.
	ErrTasksRequired = kernel.Invalid("el plan de una etapa debe vincular al menos una tarea")
	// ErrBugPlanTasks indica que un plan de bug trae tareas.
	ErrBugPlanTasks = kernel.Invalid("el plan de un bug no vincula tareas")
	// ErrTaskRepeated indica que una tarea viene repetida.
	ErrTaskRepeated = kernel.Invalid("una tarea viene repetida")
	// ErrTaskOutsideStage indica que la tarea es de otra etapa.
	ErrTaskOutsideStage = kernel.NotFound("la tarea no pertenece a la etapa del plan")
	// ErrVersionReasonRequired indica que la versión nueva no trae motivo.
	ErrVersionReasonRequired = kernel.Invalid("el motivo de la versión es obligatorio")
	// ErrListModeInvalid indica que la consulta no pide exactamente un filtro.
	ErrListModeInvalid = kernel.Invalid("indica stage_id, task_id o bug_id: exactamente uno")
	// ErrDPOnly indica que el perfil no gestiona planes de desarrollo.
	ErrDPOnly = kernel.Forbidden("solo DP gestiona planes de desarrollo")
)

// Content es lo que define una versión del plan. Todos sus campos son obligatorios:
// si uno no aplica, se escribe el motivo.
type Content struct {
	Scope              string
	TechnicalDecisions string
	ScreenStates       string
	EdgeCases          string
	ErrorHandling      string
	AcceptanceCriteria string
}

// normalized recorta el contenido y exige todos sus campos.
func (c Content) normalized() (Content, error) {
	fields := []struct {
		name  string
		value *string
	}{
		{"scope", &c.Scope},
		{"technical_decisions", &c.TechnicalDecisions},
		{"screen_states", &c.ScreenStates},
		{"edge_cases", &c.EdgeCases},
		{"error_handling", &c.ErrorHandling},
		{"acceptance_criteria", &c.AcceptanceCriteria},
	}
	for _, f := range fields {
		*f.value = strings.TrimSpace(*f.value)
		if *f.value == "" {
			return Content{}, kernel.Invalid(f.name + " es obligatorio: si no aplica, escribe el motivo")
		}
	}
	return c, nil
}

// Version es una versión inmutable de un plan.
type Version struct {
	Version   int
	Content   Content
	Reason    string
	CreatedBy string
	CreatedAt time.Time
}

// Plan es la raíz del agregado de un plan de desarrollo. Pertenece a una etapa, con
// sus tareas, o a un bug, y los referencia por ID, sin FK. No tiene status ni
// aprobación: vale Latest, su última versión.
type Plan struct {
	ID        uuid.UUID
	StageID   *uuid.UUID
	BugID     *uuid.UUID
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
	Latest    Version
	TaskIDs   []uuid.UUID
}

// EnsureDP comprueba que el perfil gestione planes.
func EnsureDP(actor kernel.Actor) error {
	if actor.Profile != kernel.ProfileDP {
		return ErrDPOnly
	}
	return nil
}

// ensureTasks comprueba las tareas según a quién pertenece el plan.
func ensureTasks(forBug bool, taskIDs []uuid.UUID) error {
	if forBug {
		if len(taskIDs) > 0 {
			return ErrBugPlanTasks
		}
		return nil
	}
	if len(taskIDs) == 0 {
		return ErrTasksRequired
	}
	seen := make(map[uuid.UUID]bool, len(taskIDs))
	for _, id := range taskIDs {
		if seen[id] {
			return ErrTaskRepeated
		}
		seen[id] = true
	}
	return nil
}

// NewPlan crea un plan con su versión 1. Exactamente uno de stageID y bugID.
func NewPlan(stageID, bugID *uuid.UUID, taskIDs []uuid.UUID, content Content, actor kernel.Actor) (Plan, error) {
	if err := EnsureDP(actor); err != nil {
		return Plan{}, err
	}
	if (stageID == nil) == (bugID == nil) {
		return Plan{}, ErrModeInvalid
	}
	if err := ensureTasks(bugID != nil, taskIDs); err != nil {
		return Plan{}, err
	}
	content, err := content.normalized()
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		ID:        uuid.New(),
		StageID:   stageID,
		BugID:     bugID,
		CreatedBy: actor.Name,
		Latest:    Version{Version: 1, Content: content, CreatedBy: actor.Name},
		TaskIDs:   taskIDs,
	}, nil
}

// Revise crea una versión nueva, con motivo, y reemplaza las tareas. Las anteriores
// se conservan.
func (p *Plan) Revise(content Content, reason string, taskIDs []uuid.UUID, actor kernel.Actor) error {
	if err := EnsureDP(actor); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrVersionReasonRequired
	}
	if err := ensureTasks(p.BugID != nil, taskIDs); err != nil {
		return err
	}
	content, err := content.normalized()
	if err != nil {
		return err
	}
	p.Latest = Version{Version: p.Latest.Version + 1, Content: content, Reason: reason, CreatedBy: actor.Name}
	p.TaskIDs = taskIDs
	return nil
}
