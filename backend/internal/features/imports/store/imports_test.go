package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/imports/domain"
	projectdomain "pmt/internal/features/projects/domain"
	projectstore "pmt/internal/features/projects/store"
	stagedomain "pmt/internal/features/stages/domain"
	stagestore "pmt/internal/features/stages/store"
	taskdomain "pmt/internal/features/tasks/domain"
	"pmt/internal/platform/postgres"
)

// TestSaveLeavesNothingOnFailure necesita PostgreSQL: se omite sin TEST_DATABASE_URL.
// La última tarea apunta a una etapa inexistente, así que falla la llave foránea
// después de insertar el proyecto, la etapa y la primera tarea.
func TestSaveLeavesNothingOnFailure(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida")
	}
	ctx := context.Background()
	pool, err := postgres.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	project, err := projectdomain.NewProject("Atomicidad", "Prueba")
	if err != nil {
		t.Fatal(err)
	}
	stage, err := stagedomain.NewStage(project.ID, "Etapa", "Prueba")
	if err != nil {
		t.Fatal(err)
	}
	valid, err := taskdomain.NewTask(stage.ID, "Válida", "Prueba")
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := taskdomain.NewTask(uuid.New(), "Huérfana", "Prueba")
	if err != nil {
		t.Fatal(err)
	}

	err = NewImports(pool).Save(ctx, domain.Plan{
		Project: project,
		Stages:  []domain.StagePlan{{Stage: stage, Tasks: []taskdomain.Task{valid, orphan}}},
	})

	if !errors.Is(err, taskdomain.ErrStageNotFound) {
		t.Fatalf("err = %v", err)
	}
	if _, err := projectstore.NewProjects(pool).Get(ctx, project.ID); !errors.Is(err, projectdomain.ErrProjectNotFound) {
		t.Errorf("proyecto: err = %v", err)
	}
	if _, err := stagestore.NewStages(pool).Get(ctx, stage.ID); !errors.Is(err, stagedomain.ErrStageNotFound) {
		t.Errorf("etapa: err = %v", err)
	}
}
