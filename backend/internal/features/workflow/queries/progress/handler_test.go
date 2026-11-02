package progress

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	bugs "pmt/internal/features/bugs/domain"
	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

var base = time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

type fakeReader struct {
	project    projects.Project
	projectErr error
	stages     []stages.Stage
	tasks      map[uuid.UUID][]tasks.Task
	bugs       []bugs.Bug
}

func (r *fakeReader) Project(context.Context, uuid.UUID) (projects.Project, error) {
	return r.project, r.projectErr
}

func (r *fakeReader) Stages(context.Context, uuid.UUID) ([]stages.Stage, error) {
	return r.stages, nil
}

func (r *fakeReader) Tasks(_ context.Context, stageID uuid.UUID) ([]tasks.Task, error) {
	return r.tasks[stageID], nil
}

func (r *fakeReader) Bugs(context.Context, uuid.UUID) ([]bugs.Bug, error) {
	return r.bugs, nil
}

// reader arma un proyecto con una etapa por cada lista de status de tareas.
func reader(project projects.Status, list ...stageFixture) *fakeReader {
	r := &fakeReader{
		project: projects.Project{ID: uuid.New(), Name: "P", Status: project, UpdatedAt: base},
		tasks:   map[uuid.UUID][]tasks.Task{},
	}
	for i, f := range list {
		s := stages.Stage{ID: uuid.New(), Name: f.name, Position: i + 1, Status: f.status, UpdatedAt: base}
		r.stages = append(r.stages, s)
		for _, status := range f.tasks {
			r.tasks[s.ID] = append(r.tasks[s.ID], tasks.Task{ID: uuid.New(), StageID: s.ID, Title: "T", Status: status, UpdatedAt: base})
		}
	}
	return r
}

type stageFixture struct {
	name   string
	status stages.Status
	tasks  []tasks.Status
}

func codes(signals []Signal) []string {
	out := []string{}
	for _, s := range signals {
		out = append(out, s.Code)
	}
	return out
}

func TestSignals(t *testing.T) {
	const (
		ready   = tasks.StatusReady
		done    = tasks.StatusDone
		cancel  = tasks.StatusCancelled
		backlog = tasks.StatusBacklog
	)
	inDev := stages.StatusInDev
	standBy := stages.StatusStandBy
	closed := stages.StatusDone

	cases := []struct {
		name    string
		project projects.Status
		stages  []stageFixture
		want    []string
	}{
		{"prueba real: 1 en done y 2 en ready", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{done, ready, ready}}}, []string{SignalReadyWithoutWork}},
		{"ready con trabajo en curso", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{ready, tasks.StatusInDev}}}, nil},
		{"ready con revisión", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{ready, tasks.StatusInReview}}}, nil},
		{"ready con corrección", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{ready, tasks.StatusChangesRequested}}}, nil},
		{"ready con QA", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{ready, tasks.StatusInQA}}}, nil},
		{"ready con cierre pendiente", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{ready, tasks.StatusPendingClose}}}, nil},
		{"ready en etapa detenida", projects.StatusInDev, []stageFixture{{"A", stages.StatusStopped, []tasks.Status{ready}}}, nil},
		{"tareas terminadas sin cerrar", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{done, cancel}}}, []string{SignalStageNotClosed}},
		{"todas canceladas no cierra", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{cancel}}}, nil},
		{"etapa vacía", projects.StatusInDev, []stageFixture{{"A", inDev, nil}}, nil},
		{"falta una en backlog", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{done, backlog}}}, nil},
		{"etapa sin iniciar con proyecto en stand_by", projects.StatusStandBy, []stageFixture{{"A", standBy, []tasks.Status{backlog}}}, []string{SignalStageNotStarted}},
		{"etapa sin iniciar con proyecto en in_dev", projects.StatusInDev, []stageFixture{{"A", closed, []tasks.Status{done}}, {"B", standBy, []tasks.Status{backlog}}}, []string{SignalStageNotStarted}},
		{"etapa sin iniciar con proyecto en working", projects.StatusWorking, []stageFixture{{"A", standBy, []tasks.Status{backlog}}}, []string{SignalStageNotStarted}},
		{"otra etapa en in_dev", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{tasks.StatusInDev}}, {"B", standBy, []tasks.Status{backlog}}}, nil},
		{"proyecto detenido", projects.StatusStopped, []stageFixture{{"A", standBy, []tasks.Status{backlog}}}, nil},
		{"ready con proyecto detenido", projects.StatusStopped, []stageFixture{{"A", inDev, []tasks.Status{ready}}}, nil},
		{"ready con proyecto cancelado", projects.StatusCancelled, []stageFixture{{"A", inDev, []tasks.Status{ready}}}, nil},
		{"sin cerrar con proyecto detenido", projects.StatusStopped, []stageFixture{{"A", inDev, []tasks.Status{done}}}, nil},
		{"sin cerrar con proyecto cancelado", projects.StatusCancelled, []stageFixture{{"A", inDev, []tasks.Status{done}}}, nil},
		{"proyecto en planning", projects.StatusPlanning, []stageFixture{{"A", stages.StatusPlanning, []tasks.Status{backlog}}}, nil},
		{"sin señales", projects.StatusInDev, []stageFixture{{"A", inDev, []tasks.Status{tasks.StatusInQA, backlog}}}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := New(reader(c.project, c.stages...)).Handle(context.Background(), uuid.New())
			if err != nil {
				t.Fatal(err)
			}
			got := codes(result.Signals)
			if len(got) != len(c.want) {
				t.Fatalf("señales = %v, quiero %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("señales = %v, quiero %v", got, c.want)
				}
			}
		})
	}
}

func TestSignalNamesFirstStandByStage(t *testing.T) {
	r := reader(projects.StatusStandBy,
		stageFixture{"A", stages.StatusStandBy, nil},
		stageFixture{"B", stages.StatusStandBy, nil})

	result, err := New(r).Handle(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Signals) != 1 || result.Signals[0].StageID != r.stages[0].ID || result.Signals[0].StageName != "A" {
		t.Errorf("señales = %+v", result.Signals)
	}
}

func TestHandleBuildsProgress(t *testing.T) {
	r := reader(projects.StatusInDev, stageFixture{"A", stages.StatusInDev, []tasks.Status{tasks.StatusReady, tasks.StatusReady, tasks.StatusInDev}})
	r.stages[0].Lead = "tl"
	r.stages[0].Members = []string{"dev"}
	later := base.Add(time.Hour)
	r.tasks[r.stages[0].ID][2].UpdatedAt = later
	r.tasks[r.stages[0].ID][2].Assignee = "dev"

	result, err := New(r).Handle(context.Background(), r.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Project.LastChangeAt.Equal(later) {
		t.Errorf("last_change_at = %v", result.Project.LastChangeAt)
	}
	stage := result.Stages[0]
	if stage.Lead != "tl" || len(stage.Members) != 1 || stage.Counts["ready"] != 2 || stage.Counts["in_dev"] != 1 || len(stage.Tasks) != 3 {
		t.Errorf("etapa = %+v", stage)
	}
	if stage.Tasks[2].Assignee != "dev" {
		t.Errorf("tarea = %+v", stage.Tasks[2])
	}
}

// Lo retirado no aparece porque Stages y Tasks ya lo excluyen: la salida es lo que
// devuelven, y una etapa sin tareas activas queda con listas vacías.
func TestEmptyProjectUsesEmptyLists(t *testing.T) {
	r := reader(projects.StatusPlanning, stageFixture{"A", stages.StatusPlanning, nil})

	result, err := New(r).Handle(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		OpenBugs OpenBugs `json:"open_bugs"`
		Signals  []Signal `json:"signals"`
		Stages   []struct {
			Members []string       `json:"members"`
			Counts  map[string]int `json:"counts"`
			Tasks   []Task         `json:"tasks"`
		} `json:"stages"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.OpenBugs.Counts == nil || got.OpenBugs.Items == nil {
		t.Errorf("salida = %s", raw)
	}
	if got.Signals == nil || got.Stages[0].Members == nil || got.Stages[0].Counts == nil || got.Stages[0].Tasks == nil {
		t.Errorf("salida = %s", raw)
	}
}

func TestOpenBugsExcludeTerminal(t *testing.T) {
	r := reader(projects.StatusWorking)
	for _, status := range []bugs.Status{
		bugs.StatusReported, bugs.StatusConfirmed, bugs.StatusConfirmed, bugs.StatusInDev, bugs.StatusInReview,
		bugs.StatusChangesRequested, bugs.StatusInQA, bugs.StatusPendingClose, bugs.StatusDone, bugs.StatusRejected,
	} {
		r.bugs = append(r.bugs, bugs.Bug{ID: uuid.New(), Title: string(status), Status: status, UpdatedAt: base})
	}

	result, err := New(r).Handle(context.Background(), r.project.ID)
	if err != nil {
		t.Fatal(err)
	}

	open := result.OpenBugs
	if len(open.Items) != 8 || open.Counts["confirmed"] != 2 || open.Counts["pending_close"] != 1 {
		t.Errorf("bugs = %+v", open)
	}
	if _, found := open.Counts["done"]; found {
		t.Errorf("done cuenta como abierto: %+v", open.Counts)
	}
	if _, found := open.Counts["rejected"]; found {
		t.Errorf("rejected cuenta como abierto: %+v", open.Counts)
	}
	for _, b := range open.Items {
		if !b.Status.IsOpen() {
			t.Errorf("bug terminal en la lista: %+v", b)
		}
	}
}

func TestBugsDoNotMoveLastChange(t *testing.T) {
	r := reader(projects.StatusWorking)
	r.bugs = []bugs.Bug{{ID: uuid.New(), Status: bugs.StatusReported, UpdatedAt: base.Add(time.Hour)}}

	result, err := New(r).Handle(context.Background(), r.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Project.LastChangeAt.Equal(base) {
		t.Errorf("last_change_at = %v", result.Project.LastChangeAt)
	}
}

func TestProjectNotFound(t *testing.T) {
	r := &fakeReader{projectErr: kernel.NotFound("proyecto no encontrado")}

	_, err := New(r).Handle(context.Background(), uuid.New())

	var notFound *kernel.Error
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %v", err)
	}
}

func serveREST(r *fakeReader, id string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	New(r).Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+id+"/progress", nil))
	return rec
}

func TestREST(t *testing.T) {
	r := reader(projects.StatusInDev, stageFixture{"A", stages.StatusInDev, []tasks.Status{tasks.StatusReady}})

	rec := serveREST(r, r.project.ID.String())

	if rec.Code != http.StatusOK {
		t.Fatalf("código = %d: %s", rec.Code, rec.Body)
	}
	var got Result
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Stages) != 1 || len(got.Signals) != 1 || got.Signals[0].Code != SignalReadyWithoutWork {
		t.Errorf("salida = %s", rec.Body)
	}
}

func TestRESTErrors(t *testing.T) {
	if rec := serveREST(&fakeReader{}, "abc"); rec.Code != http.StatusBadRequest {
		t.Errorf("ID inválido: código = %d", rec.Code)
	}
	r := &fakeReader{projectErr: kernel.NotFound("proyecto no encontrado")}
	if rec := serveREST(r, uuid.NewString()); rec.Code != http.StatusNotFound {
		t.Errorf("inexistente: código = %d", rec.Code)
	}
}
