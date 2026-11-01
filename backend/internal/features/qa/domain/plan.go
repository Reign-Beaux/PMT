// Package domain contiene los conceptos y reglas de la feature qa.
package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

// Status es el estado de un plan de pruebas.
type Status string

const (
	// StatusDraft indica que QAP lo sigue editando.
	StatusDraft Status = "draft"
	// StatusSubmitted indica que espera la decisión del TL.
	StatusSubmitted Status = "submitted"
	// StatusApproved indica que el TL aprobó su última versión.
	StatusApproved Status = "approved"
	// StatusReturned indica que el TL lo devolvió con correcciones.
	StatusReturned Status = "returned"
)

var (
	// ErrPlanNotFound indica que el plan no existe.
	ErrPlanNotFound = kernel.NotFound("plan de pruebas no encontrado")
	// ErrTitleRequired indica que el título está vacío.
	ErrTitleRequired = kernel.Invalid("el título del plan es obligatorio")
	// ErrTasksRequired indica que el plan no vincula ninguna tarea.
	ErrTasksRequired = kernel.Invalid("el plan debe vincular al menos una tarea")
	// ErrTaskRepeated indica que una tarea viene repetida.
	ErrTaskRepeated = kernel.Invalid("una tarea viene repetida")
	// ErrTaskOutsideStage indica que la tarea es de otra etapa.
	ErrTaskOutsideStage = kernel.NotFound("la tarea no pertenece a la etapa del plan")
	// ErrVersionReasonRequired indica que la versión nueva no trae motivo.
	ErrVersionReasonRequired = kernel.Invalid("el motivo de la versión es obligatorio")
	// ErrContentRequired indica que falta contenido para enviar el plan.
	ErrContentRequired = kernel.Invalid("happy_path y error_cases no pueden estar vacíos")
	// ErrReviewNotesRequired indica que la devolución no trae correcciones.
	ErrReviewNotesRequired = kernel.Invalid("las correcciones son obligatorias")
	// ErrPlanSubmitted indica que el plan espera decisión y no se revisa.
	ErrPlanSubmitted = kernel.Conflict("el plan está enviado: espera la decisión del TL")
	// ErrPlanNotDraft indica que solo se envía un plan en draft.
	ErrPlanNotDraft = kernel.Conflict("solo se envía un plan en draft")
	// ErrReturnedNeedsVersion indica que un plan devuelto exige una versión nueva.
	ErrReturnedNeedsVersion = kernel.Conflict("el plan devuelto exige una versión nueva: usa revise_test_plan")
	// ErrPlanNotSubmitted indica que el TL solo decide sobre un plan enviado.
	ErrPlanNotSubmitted = kernel.Conflict("el plan no está enviado")
	// ErrQAPOnly indica que el perfil no gestiona planes.
	ErrQAPOnly = kernel.Forbidden("solo QAP gestiona planes de pruebas")
	// ErrTLOnly indica que el perfil no decide sobre planes.
	ErrTLOnly = kernel.Forbidden("solo el TL de la etapa decide sobre un plan")
)

// Version es una versión inmutable del contenido de un plan.
type Version struct {
	Version    int
	HappyPath  string
	ErrorCases string
	Reason     string
	CreatedBy  string
	CreatedAt  time.Time
}

// Plan es la raíz del agregado de un plan de pruebas. Referencia su etapa y sus
// tareas por ID, sin FK. Latest es su última versión.
type Plan struct {
	ID          uuid.UUID
	StageID     uuid.UUID
	Title       string
	Status      Status
	CreatedBy   string
	ReviewNotes string
	ReviewedBy  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Latest      Version
	TaskIDs     []uuid.UUID
}

// EnsureQAP comprueba que el perfil gestione planes.
func EnsureQAP(actor kernel.Actor) error {
	if actor.Profile != kernel.ProfileQAP {
		return ErrQAPOnly
	}
	return nil
}

// EnsureTL comprueba el perfil de quien decide.
func EnsureTL(actor kernel.Actor) error {
	if actor.Profile != kernel.ProfileTL {
		return ErrTLOnly
	}
	return nil
}

// EnsureTasks comprueba que haya tareas y que ninguna se repita.
func EnsureTasks(taskIDs []uuid.UUID) error {
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

// NewPlan crea un plan en draft con su versión 1. El contenido puede estar
// incompleto: se exige al enviarlo.
func NewPlan(stageID uuid.UUID, title, happyPath, errorCases string, taskIDs []uuid.UUID, actor kernel.Actor) (Plan, error) {
	if err := EnsureQAP(actor); err != nil {
		return Plan{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return Plan{}, ErrTitleRequired
	}
	if err := EnsureTasks(taskIDs); err != nil {
		return Plan{}, err
	}
	return Plan{
		ID:        uuid.New(),
		StageID:   stageID,
		Title:     title,
		Status:    StatusDraft,
		CreatedBy: actor.Name,
		Latest:    Version{Version: 1, HappyPath: strings.TrimSpace(happyPath), ErrorCases: strings.TrimSpace(errorCases), CreatedBy: actor.Name},
		TaskIDs:   taskIDs,
	}, nil
}

// Revise crea una versión nueva, reemplaza las tareas y vuelve a draft. Conserva la
// última revisión del TL. Un plan enviado no se revisa.
func (p *Plan) Revise(happyPath, errorCases, reason string, taskIDs []uuid.UUID, actor kernel.Actor) error {
	if p.Status == StatusSubmitted {
		return ErrPlanSubmitted
	}
	if err := EnsureQAP(actor); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrVersionReasonRequired
	}
	if err := EnsureTasks(taskIDs); err != nil {
		return err
	}
	p.Latest = Version{
		Version:    p.Latest.Version + 1,
		HappyPath:  strings.TrimSpace(happyPath),
		ErrorCases: strings.TrimSpace(errorCases),
		Reason:     reason,
		CreatedBy:  actor.Name,
	}
	p.TaskIDs = taskIDs
	p.Status = StatusDraft
	return nil
}

// Submit envía el plan al TL. Un plan devuelto no se reenvía sin una versión nueva.
func (p *Plan) Submit(actor kernel.Actor) error {
	if p.Status == StatusReturned {
		return ErrReturnedNeedsVersion
	}
	if p.Status != StatusDraft {
		return ErrPlanNotDraft
	}
	if err := EnsureQAP(actor); err != nil {
		return err
	}
	if p.Latest.HappyPath == "" || p.Latest.ErrorCases == "" {
		return ErrContentRequired
	}
	p.Status = StatusSubmitted
	return nil
}

// EnsureDecidable comprueba status y perfil, antes de consultar el ámbito en otra
// feature.
func (p Plan) EnsureDecidable(actor kernel.Actor) error {
	if p.Status != StatusSubmitted {
		return ErrPlanNotSubmitted
	}
	return EnsureTL(actor)
}

// Approve aprueba la última versión y limpia las correcciones anteriores, ya
// atendidas. El ámbito se comprueba antes, en el handler.
func (p *Plan) Approve(actor kernel.Actor) error {
	if err := p.EnsureDecidable(actor); err != nil {
		return err
	}
	p.Status = StatusApproved
	p.ReviewNotes = ""
	p.ReviewedBy = actor.Name
	return nil
}

// Return devuelve el plan con correcciones obligatorias. Solo espacios cuenta como
// vacío. El ámbito se comprueba antes, en el handler.
func (p *Plan) Return(notes string, actor kernel.Actor) error {
	if err := p.EnsureDecidable(actor); err != nil {
		return err
	}
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return ErrReviewNotesRequired
	}
	p.Status = StatusReturned
	p.ReviewNotes = notes
	p.ReviewedBy = actor.Name
	return nil
}
