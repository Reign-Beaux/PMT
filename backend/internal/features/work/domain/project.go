// Package domain contiene los conceptos y reglas del módulo work.
package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrProjectNotFound indica que el proyecto no existe.
	ErrProjectNotFound = errors.New("proyecto no encontrado")
	// ErrProjectNameRequired indica que el nombre del proyecto está vacío.
	ErrProjectNameRequired = errors.New("el nombre del proyecto es obligatorio")
)

// Project es la raíz del agregado de un proyecto.
type Project struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewProject crea un proyecto válido.
func NewProject(name, description string) (Project, error) {
	p := Project{ID: uuid.New()}
	if err := p.Update(name, description); err != nil {
		return Project{}, err
	}
	return p, nil
}

// Update cambia los datos editables del proyecto.
func (p *Project) Update(name, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrProjectNameRequired
	}
	p.Name = name
	p.Description = strings.TrimSpace(description)
	return nil
}
