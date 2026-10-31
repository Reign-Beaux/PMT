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
	"pmt/internal/kernel"
)

const validFile = `
stage:
  name: "Pagos"
  description: "Cobro con tarjeta."
  tasks:
    - title: "Tabla de pagos"
      description: |
        Se relaciona con pedidos.
    - title: "Cobrar"
      description: "Con tarjeta."
`

type fakeRepo struct {
	saved []domain.StagePlan
	err   error
}

func (r *fakeRepo) SaveStage(_ context.Context, _ uuid.UUID, plan domain.StagePlan) error {
	if r.err != nil {
		return r.err
	}
	r.saved = append(r.saved, plan)
	return nil
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "etapa.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func command(t *testing.T, content string) Command {
	t.Helper()
	return Command{ProjectID: uuid.New(), Path: writeFile(t, content), Reason: "Petición", RequestedBy: "PM"}
}

func TestHandleCreatesStageInOwnPlanning(t *testing.T) {
	repo := &fakeRepo{}
	cmd := command(t, validFile)

	result, err := New(repo).Handle(context.Background(), cmd)

	if err != nil {
		t.Fatal(err)
	}
	if result.StageName != "Pagos" || result.TaskCount != 2 || result.ProjectID != cmd.ProjectID {
		t.Errorf("result = %+v", result)
	}
	plan := repo.saved[0]
	if !plan.Stage.InOwnPlanning() || plan.Stage.ID != result.StageID || plan.Stage.ProjectID != cmd.ProjectID {
		t.Errorf("etapa = %+v", plan.Stage)
	}
	if plan.Tasks[1].Title != "Cobrar" || plan.Tasks[1].StageID != plan.Stage.ID {
		t.Errorf("tareas = %+v", plan.Tasks)
	}
}

func TestHandleRejectsInvalidFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"sin nombre", "stage:\n  description: B\n", "etapa: el nombre de la etapa es obligatorio"},
		{"sin raíz", "otra: 1\n", `línea 1: clave desconocida "otra"`},
		{"raíz de proyecto", "project:\n  name: A\n", `línea 1: clave desconocida "project"`},
		{"sin descripción", "stage:\n  name: A\n", "etapa: la descripción es obligatoria"},
		{"tarea sin título", "stage:\n  name: A\n  description: B\n  tasks:\n    - description: C\n", "tarea 1: el título de la tarea es obligatorio"},
		{"tarea sin descripción", "stage:\n  name: A\n  description: B\n  tasks:\n    - title: C\n", "tarea 1: la descripción es obligatoria"},
		{"comentario en etapa", "stage:\n  name: A # nota\n  description: B\n", "etapa: el texto contiene un comentario; ponlo entre comillas"},
		{"comentario en tarea", "stage:\n  name: A\n  description: B\n  tasks:\n    - title: C # nota\n      description: D\n", "tarea 1: el texto contiene un comentario; ponlo entre comillas"},
		{"vacío", "", "el fichero está vacío"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

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

func TestHandleRequiresReasonAndRequester(t *testing.T) {
	cmd := command(t, validFile)
	cmd.Reason = " "

	_, err := New(&fakeRepo{}).Handle(context.Background(), cmd)

	if err == nil || err.Error() != "etapa: el motivo del cambio es obligatorio" {
		t.Errorf("err = %v", err)
	}
}

func TestHandleReturnsRepositoryError(t *testing.T) {
	errNotWorking := kernel.Conflict("el proyecto no está en working")

	_, err := New(&fakeRepo{err: errNotWorking}).Handle(context.Background(), command(t, validFile))

	if !errors.Is(err, errNotWorking) {
		t.Errorf("err = %v", err)
	}
}

func TestHandleReportsUnreadableFile(t *testing.T) {
	cmd := command(t, validFile)
	cmd.Path += ".falta"

	_, err := New(&fakeRepo{}).Handle(context.Background(), cmd)

	if err == nil || !strings.HasPrefix(err.Error(), "no se pudo leer el fichero") {
		t.Errorf("err = %v", err)
	}
}
