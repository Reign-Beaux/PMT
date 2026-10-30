// Package usecase contiene las operaciones de negocio del módulo work.
package usecase

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/work/domain"
)

// ProjectRepository es el puerto de persistencia de proyectos.
type ProjectRepository interface {
	Create(ctx context.Context, p domain.Project) (domain.Project, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Project, error)
	List(ctx context.Context) ([]domain.Project, error)
	Update(ctx context.Context, p domain.Project) (domain.Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Projects agrupa los casos de uso de proyectos.
type Projects struct {
	repo ProjectRepository
}

// NewProjects crea los casos de uso de proyectos.
func NewProjects(repo ProjectRepository) *Projects {
	return &Projects{repo: repo}
}

// Create registra un proyecto nuevo.
func (u *Projects) Create(ctx context.Context, name, description string) (domain.Project, error) {
	p, err := domain.NewProject(name, description)
	if err != nil {
		return domain.Project{}, err
	}
	return u.repo.Create(ctx, p)
}

// Get devuelve un proyecto.
func (u *Projects) Get(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	return u.repo.Get(ctx, id)
}

// List devuelve todos los proyectos.
func (u *Projects) List(ctx context.Context) ([]domain.Project, error) {
	return u.repo.List(ctx)
}

// Update cambia los datos de un proyecto.
func (u *Projects) Update(ctx context.Context, id uuid.UUID, name, description string) (domain.Project, error) {
	p, err := u.repo.Get(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}
	if err := p.Update(name, description); err != nil {
		return domain.Project{}, err
	}
	return u.repo.Update(ctx, p)
}

// Delete elimina un proyecto.
func (u *Projects) Delete(ctx context.Context, id uuid.UUID) error {
	return u.repo.Delete(ctx, id)
}
