// Package domain contiene los conceptos y reglas de la feature bugs.
package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrBugNotFound indica que el bug no existe.
	ErrBugNotFound = kernel.NotFound("bug no encontrado")
	// ErrBugTitleRequired indica que el título del bug está vacío.
	ErrBugTitleRequired = kernel.Invalid("el título del bug es obligatorio")
	// ErrBugStepsRequired indica que faltan los pasos para reproducirlo.
	ErrBugStepsRequired = kernel.Invalid("los pasos para reproducir el bug son obligatorios")
	// ErrBugVersionRequired indica que falta la versión donde apareció.
	ErrBugVersionRequired = kernel.Invalid("la versión del producto es obligatoria")
	// ErrProjectNotFound indica que el proyecto del bug no existe.
	ErrProjectNotFound = kernel.NotFound("proyecto no encontrado")
	// ErrBugClosed indica que el bug ya terminó y su reporte no se corrige.
	ErrBugClosed = kernel.Conflict("un bug en done o rejected no se edita")
)

// Bug es la raíz del agregado de un bug. Referencia a su proyecto por ID y no
// pertenece a ninguna etapa.
type Bug struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	Title            string
	Description      string
	StepsToReproduce string
	ProductVersion   string
	Status           Status
	ReportedBy       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewBug crea un bug válido en su estado inicial. reportedBy es quien lo registra;
// los bugs anteriores a esta columna no tienen autor.
func NewBug(projectID uuid.UUID, title, description, steps, productVersion, reportedBy string) (Bug, error) {
	b := Bug{ID: uuid.New(), ProjectID: projectID, Status: StatusReported, ReportedBy: reportedBy}
	if err := b.Update(title, description, steps, productVersion); err != nil {
		return Bug{}, err
	}
	return b, nil
}

// Update cambia los datos editables del bug, salvo si ya terminó.
func (b *Bug) Update(title, description, steps, productVersion string) error {
	if !b.Status.IsOpen() {
		return ErrBugClosed
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrBugTitleRequired
	}
	steps = strings.TrimSpace(steps)
	if steps == "" {
		return ErrBugStepsRequired
	}
	productVersion = strings.TrimSpace(productVersion)
	if productVersion == "" {
		return ErrBugVersionRequired
	}
	b.Title = title
	b.Description = strings.TrimSpace(description)
	b.StepsToReproduce = steps
	b.ProductVersion = productVersion
	return nil
}
