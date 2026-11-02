// Package domain contiene los conceptos y reglas de la feature projects.
package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrProjectNotFound indica que el proyecto no existe.
	ErrProjectNotFound = kernel.NotFound("proyecto no encontrado")
	// ErrProjectNameRequired indica que el nombre del proyecto está vacío.
	ErrProjectNameRequired = kernel.Invalid("el nombre del proyecto es obligatorio")
	// ErrProjectHasStages indica que el proyecto no se puede eliminar porque tiene etapas, incluidas las retiradas, o bugs.
	ErrProjectHasStages = kernel.Conflict("el proyecto tiene etapas o bugs, activos o retirados")
	// ErrProjectNotPlanning indica que el proyecto ya no admite cambios de definición.
	ErrProjectNotPlanning = kernel.Conflict("el proyecto no está en planning")
	// ErrProjectNotWorking indica que el proyecto no admite trabajo nuevo.
	ErrProjectNotWorking = kernel.Conflict("el proyecto no está en working")
	// ErrProductVersionFrozen indica que la versión de producto ya no se puede cambiar.
	ErrProductVersionFrozen = kernel.Conflict("la versión de producto no se edita fuera de planning")
)

// Project es la raíz del agregado de un proyecto.
type Project struct {
	ID             uuid.UUID
	Name           string
	Description    string
	Status         Status
	ProductVersion string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewProject crea un proyecto válido en su estado inicial y sin versión de producto.
func NewProject(name, description string) (Project, error) {
	p := Project{ID: uuid.New(), Status: StatusPlanning}
	if err := p.Update(name, description, ""); err != nil {
		return Project{}, err
	}
	return p, nil
}

// EnsurePlanning devuelve un error si el proyecto no está en planning.
func (p Project) EnsurePlanning() error {
	if p.Status != StatusPlanning {
		return ErrProjectNotPlanning
	}
	return nil
}

// IsWorking indica si el proyecto admite trabajo nuevo.
func (p Project) IsWorking() bool {
	return p.Status == StatusWorking
}

// AcceptsWork indica si sus etapas y tareas pueden avanzar.
func (p Project) AcceptsWork() bool {
	return p.Status == StatusInDev || p.Status == StatusWorking
}

// IsFrozen indica si el proyecto congela sus etapas y tareas.
func (p Project) IsFrozen() bool {
	return p.Status == StatusStopped || p.Status == StatusCancelled
}

// Update cambia los datos editables del proyecto. Fuera de planning, la versión
// de producto debe ser la actual.
func (p *Project) Update(name, description, productVersion string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrProjectNameRequired
	}
	productVersion = strings.TrimSpace(productVersion)
	if p.Status != StatusPlanning && productVersion != p.ProductVersion {
		return ErrProductVersionFrozen
	}
	p.Name = name
	p.Description = strings.TrimSpace(description)
	p.ProductVersion = productVersion
	return nil
}
