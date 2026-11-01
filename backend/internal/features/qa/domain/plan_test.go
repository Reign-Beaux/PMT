package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	qap = kernel.Actor{Name: "qap", Profile: kernel.ProfileQAP}
	tl  = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	dev = kernel.Actor{Name: "dev", Profile: kernel.ProfileDevBackend}
)

func plan(status Status) Plan {
	return Plan{ID: uuid.New(), Status: status, Latest: Version{Version: 1, HappyPath: "H", ErrorCases: "E"}}
}

func TestNewPlan(t *testing.T) {
	task := uuid.New()
	tests := []struct {
		name  string
		title string
		tasks []uuid.UUID
		actor kernel.Actor
		want  error
	}{
		{"válido", " Plan ", []uuid.UUID{task}, qap, nil},
		{"perfil sin permiso", "Plan", []uuid.UUID{task}, tl, ErrQAPOnly},
		{"usuario sin permiso", "Plan", []uuid.UUID{task}, kernel.Owner, ErrQAPOnly},
		{"sin título", "  ", []uuid.UUID{task}, qap, ErrTitleRequired},
		{"sin tareas", "Plan", nil, qap, ErrTasksRequired},
		{"tarea repetida", "Plan", []uuid.UUID{task, task}, qap, ErrTaskRepeated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPlan(uuid.New(), tt.title, "", "", tt.tasks, tt.actor)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if err == nil && (p.Status != StatusDraft || p.Latest.Version != 1 || p.Title != "Plan") {
				t.Errorf("plan = %+v", p)
			}
		})
	}
}

func TestRevise(t *testing.T) {
	tasks := []uuid.UUID{uuid.New()}
	tests := []struct {
		name   string
		status Status
		reason string
		tasks  []uuid.UUID
		actor  kernel.Actor
		want   error
	}{
		{"desde draft", StatusDraft, "M", tasks, qap, nil},
		{"desde returned", StatusReturned, "M", tasks, qap, nil},
		{"desde approved", StatusApproved, "M", tasks, qap, nil},
		{"desde submitted", StatusSubmitted, "M", tasks, qap, ErrPlanSubmitted},
		{"perfil sin permiso", StatusDraft, "M", tasks, tl, ErrQAPOnly},
		{"sin motivo", StatusDraft, "  ", tasks, qap, ErrVersionReasonRequired},
		{"sin tareas", StatusDraft, "M", nil, qap, ErrTasksRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := plan(tt.status)
			err := p.Revise("H2", "E2", tt.reason, tt.tasks, tt.actor)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if err != nil {
				if p.Status != tt.status || p.Latest.Version != 1 {
					t.Errorf("cambió al rechazarse: %+v", p)
				}
				return
			}
			if p.Status != StatusDraft || p.Latest.Version != 2 || p.Latest.HappyPath != "H2" || len(p.TaskIDs) != 1 {
				t.Errorf("plan = %+v", p)
			}
		})
	}
}

func TestSubmit(t *testing.T) {
	empty := plan(StatusDraft)
	empty.Latest.ErrorCases = ""
	tests := []struct {
		name  string
		plan  Plan
		actor kernel.Actor
		want  error
	}{
		{"válido", plan(StatusDraft), qap, nil},
		{"perfil sin permiso", plan(StatusDraft), dev, ErrQAPOnly},
		{"contenido vacío", empty, qap, ErrContentRequired},
		{"devuelto exige versión", plan(StatusReturned), qap, ErrReturnedNeedsVersion},
		{"ya enviado", plan(StatusSubmitted), qap, ErrPlanNotDraft},
		{"aprobado", plan(StatusApproved), qap, ErrPlanNotDraft},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.plan
			err := p.Submit(tt.actor)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if err == nil && p.Status != StatusSubmitted {
				t.Errorf("status = %s", p.Status)
			}
		})
	}
}

func TestApprove(t *testing.T) {
	p := plan(StatusSubmitted)
	p.ReviewNotes = "viejas"
	if err := p.Approve(tl); err != nil {
		t.Fatal(err)
	}
	if p.Status != StatusApproved || p.ReviewedBy != "tl" || p.ReviewNotes != "" {
		t.Errorf("plan = %+v", p)
	}

	rejected := plan(StatusDraft)
	if err := rejected.Approve(tl); !errors.Is(err, ErrPlanNotSubmitted) {
		t.Errorf("sin enviar: err = %v", err)
	}
	wrong := plan(StatusSubmitted)
	if err := wrong.Approve(qap); !errors.Is(err, ErrTLOnly) {
		t.Errorf("perfil: err = %v", err)
	}
}

func TestReturn(t *testing.T) {
	p := plan(StatusSubmitted)
	if err := p.Return(" Falta X ", tl); err != nil {
		t.Fatal(err)
	}
	if p.Status != StatusReturned || p.ReviewNotes != "Falta X" || p.ReviewedBy != "tl" {
		t.Errorf("plan = %+v", p)
	}

	empty := plan(StatusSubmitted)
	if err := empty.Return("  ", tl); !errors.Is(err, ErrReviewNotesRequired) {
		t.Errorf("sin correcciones: err = %v", err)
	}
	wrong := plan(StatusSubmitted)
	if err := wrong.Return("X", kernel.Owner); !errors.Is(err, ErrTLOnly) {
		t.Errorf("perfil: err = %v", err)
	}
	approved := plan(StatusApproved)
	if err := approved.Return("X", tl); !errors.Is(err, ErrPlanNotSubmitted) {
		t.Errorf("aprobado: err = %v", err)
	}
}
