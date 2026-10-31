package domain

// Status es el estado de un bug. Por ahora solo son definiciones: no hay
// transiciones ni comando para cambiarlo.
type Status string

const (
	// StatusReported es el estado inicial: reportado, aún sin confirmar.
	StatusReported Status = "reported"
	// StatusConfirmed indica que se reprodujo y falta corregirlo.
	StatusConfirmed Status = "confirmed"
	// StatusInProgress indica que está en corrección.
	StatusInProgress Status = "in_progress"
	// StatusInReview indica que está en revisión de código.
	StatusInReview Status = "in_review"
	// StatusChangesRequested indica que tiene correcciones pendientes.
	StatusChangesRequested Status = "changes_requested"
	// StatusInQA indica que está en verificación de QA.
	StatusInQA Status = "in_qa"
	// StatusDone indica que está corregido y verificado.
	StatusDone Status = "done"
	// StatusRejected indica que no se corregirá: no se reproduce, está duplicado o no es un bug.
	StatusRejected Status = "rejected"
)
