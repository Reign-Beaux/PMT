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
	// ErrProjectHasStages indica que el proyecto no se puede eliminar porque tiene etapas.
	ErrProjectHasStages = kernel.Conflict("el proyecto tiene etapas")
	// ErrProjectNotPlanning indica que el proyecto ya no admite cambios de definición.
	ErrProjectNotPlanning = kernel.Conflict("el proyecto no está en planning")
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

// Update cambia los datos editables del proyecto.
func (p *Project) Update(name, description, productVersion string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrProjectNameRequired
	}
	p.Name = name
	p.Description = strings.TrimSpace(description)
	p.ProductVersion = strings.TrimSpace(productVersion)
	return nil
}
