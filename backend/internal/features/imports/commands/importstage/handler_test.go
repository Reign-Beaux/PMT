package importstage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/imports/domain"
	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

const validFile = `
stages:
  - name: "Pagos"
    description: "Cobro con tarjeta."
    tasks:
      - title: "Tabla de pagos"
        description: |
          Se relaciona con pedidos.
      - title: "Cobrar"
        description: "Con tarjeta."
  - name: "Reportes"
    description: "Sin tareas."
`

type fakeRepo struct {
	status projects.Status
	err    error

	gotProjectID uuid.UUID
	saved        []domain.StagePlan
}

func (r *fakeRepo) ProjectStatus(_ context.Context, projectID uuid.UUID) (projects.Status, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.status, nil
}

func (r *fakeRepo) SaveStages(_ context.Context, projectID uuid.UUID, plans []domain.StagePlan) error {
	if r.err != nil {
		return r.err
	}
	r.gotProjectID = projectID
	r.saved = append(r.saved, plans...)
	return nil
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "etapas.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func command(t *testing.T, content string) Command {
	t.Helper()
	return Command{Actor: kernel.Owner, ProjectID: uuid.New(), Path: writeFile(t, content)}
}

func TestHandlePlanningCreatesPlainStages(t *testing.T) {
	repo := &fakeRepo{status: projects.StatusPlanning}
	cmd := command(t, validFile)

	result, err := New(repo).Handle(context.Background(), cmd)

	if err != nil {
		t.Fatal(err)
	}
	if len(repo.saved) != 2 {
		t.Fatalf("guardadas = %d", len(repo.saved))
	}
	for _, plan := range repo.saved {
		if plan.Stage.InOwnPlanning() || plan.Stage.ProjectID != cmd.ProjectID || plan.Stage.AddReason != "" {
			t.Errorf("etapa = %+v", plan.Stage)
		}
	}
	if result.ProjectID != cmd.ProjectID || len(result.Stages) != 2 {
		t.Fatalf("result = %+v", result)
	}
	if result.Stages[0].StageName != "Pagos" || result.Stages[0].TaskCount != 2 || result.Stages[0].StageID != repo.saved[0].Stage.ID {
		t.Errorf("primera = %+v", result.Stages[0])
	}
	if result.Stages[1].StageName != "Reportes" || result.Stages[1].TaskCount != 0 {
		t.Errorf("segunda = %+v", result.Stages[1])
	}
	if repo.saved[0].Tasks[1].Title != "Cobrar" || repo.saved[0].Tasks[1].StageID != repo.saved[0].Stage.ID {
		t.Errorf("tareas = %+v", repo.saved[0].Tasks)
	}
}

func TestHandlePlanningIgnoresReasonAndRequester(t *testing.T) {
	repo := &fakeRepo{status: projects.StatusPlanning}
	cmd := command(t, validFile)
	cmd.Reason = "Petición"
	cmd.RequestedBy = "PM"

	if _, err := New(repo).Handle(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	for _, plan := range repo.saved {
		if plan.Stage.InOwnPlanning() {
			t.Errorf("etapa = %+v", plan.Stage)
		}
	}
}

func TestHandleWorkingCreatesOwnPlanningStages(t *testing.T) {
	repo := &fakeRepo{status: projects.StatusWorking}
	cmd := command(t, validFile)
	cmd.Reason = "Petición"
	cmd.RequestedBy = "PM"

	if _, err := New(repo).Handle(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if len(repo.saved) != 2 {
		t.Fatalf("guardadas = %d", len(repo.saved))
	}
	for _, plan := range repo.saved {
		if !plan.Stage.InOwnPlanning() || plan.Stage.AddReason != "Petición" || plan.Stage.AddRequestedBy != "PM" {
			t.Errorf("etapa = %+v", plan.Stage)
		}
	}
}

func TestHandleWorkingRequiresReasonAndRequester(t *testing.T) {
	tests := []struct {
		name        string
		reason      string
		requestedBy string
		want        error
	}{
		{"sin motivo", " ", "PM", stages.ErrChangeReasonRequired},
		{"sin solicitante", "Petición", " ", stages.ErrChangeRequesterRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{status: projects.StatusWorking}
			cmd := command(t, validFile)
			cmd.Reason, cmd.RequestedBy = tt.reason, tt.requestedBy

			_, err := New(repo).Handle(context.Background(), cmd)

			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, se esperaba %v", err, tt.want)
			}
			if len(repo.saved) != 0 {
				t.Error("no debió guardar nada")
			}
		})
	}
}

func TestHandleRejectsOtherStatus(t *testing.T) {
	for _, status := range []projects.Status{projects.StatusInDev, projects.StatusStandBy, projects.StatusStopped, projects.StatusCancelled} {
		repo := &fakeRepo{status: status}

		_, err := New(repo).Handle(context.Background(), command(t, validFile))

		if !errors.Is(err, domain.ErrProjectNotImportable) {
			t.Errorf("%s: err = %v", status, err)
		}
		if len(repo.saved) != 0 {
			t.Errorf("%s: no debió guardar nada", status)
		}
	}
}

func TestHandleRejectsInvalidFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"raíz anterior", "stage:\n  name: A\n  description: B\n", "la raíz ahora es stages, con una lista de etapas"},
		{"sin etapas", "stages: []\n", "el fichero no tiene etapas"},
		{"etapa 2 sin descripción", "stages:\n  - name: A\n    description: B\n  - name: C\n", "etapa 2: la descripción es obligatoria"},
		{"tarea sin título", "stages:\n  - name: A\n    description: B\n    tasks:\n      - description: C\n", "etapa 1, tarea 1: el título de la tarea es obligatorio"},
		{"vacío", "", "el fichero está vacío"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{status: projects.StatusPlanning}

			_, err := New(repo).Handle(context.Background(), command(t, tt.content))

			if err == nil || err.Error() != tt.want {
				t.Errorf("err = %v, se esperaba %q", err, tt.want)
			}
			if len(repo.saved) != 0 {
				t.Error("no debió guardar nada")
			}
		})
	}
}

func TestHandleRejectsNonPlanner(t *testing.T) {
	repo := &fakeRepo{status: projects.StatusPlanning}
	cmd := command(t, validFile)
	cmd.Actor = kernel.Actor{Name: "devback", Profile: kernel.ProfileDevBackend}

	_, err := New(repo).Handle(context.Background(), cmd)

	if !errors.Is(err, domain.ErrImportForbidden) {
		t.Errorf("err = %v", err)
	}
	if len(repo.saved) != 0 {
		t.Error("no debió guardar nada")
	}
}

func TestHandleReturnsRepositoryError(t *testing.T) {
	errDB := kernel.Conflict("falló la base")
	repo := &fakeRepo{status: projects.StatusPlanning, err: errDB}

	_, err := New(repo).Handle(context.Background(), command(t, validFile))

	if !errors.Is(err, errDB) {
		t.Errorf("err = %v", err)
	}
}

func TestHandleReportsUnreadableFile(t *testing.T) {
	cmd := command(t, validFile)
	cmd.Path += ".falta"

	_, err := New(&fakeRepo{status: projects.StatusPlanning}).Handle(context.Background(), cmd)

	if err == nil || !strings.HasPrefix(err.Error(), "no se pudo leer el fichero") {
		t.Errorf("err = %v", err)
	}
}
