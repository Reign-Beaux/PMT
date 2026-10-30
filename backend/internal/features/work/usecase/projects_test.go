package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/work/domain"
)

type fakeRepo struct {
	projects map[uuid.UUID]domain.Project
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{projects: map[uuid.UUID]domain.Project{}}
}

func (r *fakeRepo) Create(_ context.Context, p domain.Project) (domain.Project, error) {
	r.projects[p.ID] = p
	return p, nil
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Project, error) {
	p, ok := r.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return p, nil
}

func (r *fakeRepo) List(context.Context) ([]domain.Project, error) {
	projects := make([]domain.Project, 0, len(r.projects))
	for _, p := range r.projects {
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *fakeRepo) Update(_ context.Context, p domain.Project) (domain.Project, error) {
	r.projects[p.ID] = p
	return p, nil
}

func (r *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.projects[id]; !ok {
		return domain.ErrProjectNotFound
	}
	delete(r.projects, id)
	return nil
}

func TestCreateRejectsInvalidProject(t *testing.T) {
	repo := newFakeRepo()

	_, err := NewProjects(repo).Create(context.Background(), " ", "")

	if !errors.Is(err, domain.ErrProjectNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.projects) != 0 {
		t.Error("no debió guardar el proyecto")
	}
}

func TestUpdateChangesProject(t *testing.T) {
	ctx := context.Background()
	projects := NewProjects(newFakeRepo())
	created, err := projects.Create(ctx, "PMT", "")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := projects.Update(ctx, created.ID, "PMT v4", "Gestor")
	if err != nil {
		t.Fatal(err)
	}

	if updated.Name != "PMT v4" || updated.Description != "Gestor" {
		t.Errorf("proyecto = %+v", updated)
	}
}

func TestUpdateMissingProject(t *testing.T) {
	_, err := NewProjects(newFakeRepo()).Update(context.Background(), uuid.New(), "PMT", "")

	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("err = %v", err)
	}
}
