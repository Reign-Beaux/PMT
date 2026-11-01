package domain

import "pmt/internal/kernel"

var (
	// ErrTaskStatusInvalid indica que el status no es uno de los valores conocidos.
	ErrTaskStatusInvalid = kernel.Invalid("status desconocido: usa backlog, ready, in_dev, in_review, changes_requested, in_qa, pending_close, done o cancelled")
	// ErrTaskStatusTransition indica que el status actual no permite pasar al pedido.
	ErrTaskStatusTransition = kernel.Conflict("transición de status no permitida")
	// ErrStageNoActiveTasks aplica la regla 4: no se libera una etapa sin trabajo.
	ErrStageNoActiveTasks = kernel.Conflict("la etapa no tiene tareas activas")
	// ErrStageTasksPending aplica la decisión 8: quedan tareas sin terminar.
	ErrStageTasksPending = kernel.Conflict("la etapa no se puede cerrar: tiene tareas que no están en done o cancelled")
	// ErrStageNoTasksDone aplica la decisión 8: sin tareas en done, la etapa se cancela.
	ErrStageNoTasksDone = kernel.Conflict("la etapa no tiene tareas en done: cancélala")
	// ErrTaskDoneRevised aplica la decisión 7: lo terminado no se revisa.
	ErrTaskDoneRevised = kernel.Conflict("una tarea en done no se revisa: usa add_task")
)

// statusTransitions indica a qué estados puede pasar cada estado.
var statusTransitions = map[Status][]Status{
	StatusBacklog:          {StatusReady, StatusCancelled},
	StatusReady:            {StatusInDev, StatusCancelled},
	StatusInDev:            {StatusInReview, StatusCancelled},
	StatusInReview:         {StatusInQA, StatusChangesRequested, StatusPendingClose, StatusCancelled},
	StatusChangesRequested: {StatusInDev, StatusCancelled},
	StatusInQA:             {StatusPendingClose, StatusChangesRequested, StatusCancelled},
	StatusPendingClose:     {StatusDone, StatusInReview, StatusInQA, StatusCancelled},
	StatusDone:             {},
	StatusCancelled:        {StatusBacklog},
}

// IsKnownStatus indica si el valor es un status de tarea.
func IsKnownStatus(status Status) bool {
	_, known := statusTransitions[status]
	return known
}

// IsCancelOrReopen indica si la transición cancela o reabre la tarea en lugar de
// hacerla avanzar. Cada caso sigue sus propias reglas entre status.
func IsCancelOrReopen(from, to Status) bool {
	return to == StatusCancelled || from == StatusCancelled
}

// Closure es el recuento de las tareas de una etapa para decidir su cierre. Las
// canceladas y las retiradas no cuentan.
type Closure struct {
	Done    int
	Pending int
}

// EnsureClosable aplica la decisión 8: todas terminadas y al menos una en done.
func (c Closure) EnsureClosable() error {
	if c.Pending > 0 {
		return ErrStageTasksPending
	}
	if c.Done == 0 {
		return ErrStageNoTasksDone
	}
	return nil
}

// ReviseStatus aplica la decisión 7: da el status de la tarea tras revisar su
// contenido. Lo ya entregado vuelve a correcciones para no aprobar algo distinto
// a lo pedido, y lo terminado no se revisa.
func (t Task) ReviseStatus() (Status, error) {
	switch t.Status {
	case StatusInReview, StatusInQA, StatusPendingClose:
		return StatusChangesRequested, nil
	case StatusDone:
		return "", ErrTaskDoneRevised
	}
	return t.Status, nil
}

// ChangeStatus pasa la tarea a otro estado si la transición está permitida y
// quien actúa puede hacerla. Tomarla la deja a nombre del dev; reabrirla la libera.
func (t *Task) ChangeStatus(status Status, actor kernel.Actor, reason string, team kernel.Team) error {
	if err := t.EnsureProfile(status, actor); err != nil {
		return err
	}
	if err := t.ensureScope(actor, team); err != nil {
		return err
	}
	if err := ensureReason(t.Status, status, reason); err != nil {
		return err
	}
	switch {
	case t.Status == StatusReady && status == StatusInDev:
		t.Assignee = actor.Name
	case status == StatusBacklog:
		t.Assignee = ""
	}
	t.Status = status
	return nil
}
