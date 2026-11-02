package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	dp      = kernel.Actor{Name: "dp", Profile: kernel.ProfileDP}
	content = Content{"S", "T", "P", "E", "F", "A"}
)

func TestNewPlanModes(t *testing.T) {
	id, task := uuid.New(), uuid.New()
	tests := []struct {
		name  string
		stage *uuid.UUID
		bug   *uuid.UUID
		tasks []uuid.UUID
		want  error
	}{
		{"etapa con tarea", &id, nil, []uuid.UUID{task}, nil},
		{"bug sin tareas", nil, &id, nil, nil},
		{"ninguno", nil, nil, nil, ErrModeInvalid},
		{"ambos", &id, &id, nil, ErrModeInvalid},
		{"etapa sin tareas", &id, nil, nil, ErrTasksRequired},
		{"bug con tareas", nil, &id, []uuid.UUID{task}, ErrBugPlanTasks},
		{"tarea repetida", &id, nil, []uuid.UUID{task, task}, ErrTaskRepeated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := NewPlan(tt.stage, tt.bug, tt.tasks, content, dp)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if tt.want == nil && (plan.Latest.Version != 1 || plan.CreatedBy != "dp") {
				t.Errorf("plan = %+v", plan)
			}
		})
	}
}

func TestOnlyDPManagesPlans(t *testing.T) {
	id := uuid.New()
	plan := Plan{BugID: &id, Latest: Version{Version: 1, Content: content}}
	others := []kernel.Profile{
		kernel.ProfilePP, kernel.ProfilePC, kernel.ProfileTL, kernel.ProfileDevFrontend, kernel.ProfileDevBackend,
		kernel.ProfileCR, kernel.ProfileQAP, kernel.ProfileQA, kernel.ProfileUser,
	}
	for _, profile := range others {
		actor := kernel.Actor{Name: "otro", Profile: profile}
		if _, err := NewPlan(nil, &id, nil, content, actor); !errors.Is(err, ErrDPOnly) {
			t.Errorf("%s crea: %v", profile, err)
		}
		if err := plan.Revise(content, "motivo", nil, actor); !errors.Is(err, ErrDPOnly) {
			t.Errorf("%s revisa: %v", profile, err)
		}
	}
}

func TestContentRequired(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		field  string
		change func(*Content)
	}{
		{"scope", func(c *Content) { c.Scope = "  " }},
		{"technical_decisions", func(c *Content) { c.TechnicalDecisions = "" }},
		{"screen_states", func(c *Content) { c.ScreenStates = "" }},
		{"edge_cases", func(c *Content) { c.EdgeCases = "" }},
		{"error_handling", func(c *Content) { c.ErrorHandling = "" }},
		{"acceptance_criteria", func(c *Content) { c.AcceptanceCriteria = "\n" }},
	}
	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			incomplete := content
			tt.change(&incomplete)

			_, err := NewPlan(nil, &id, nil, incomplete, dp)
			if err == nil || !strings.HasPrefix(err.Error(), tt.field+" ") {
				t.Fatalf("crear: %v", err)
			}
			plan := Plan{BugID: &id, Latest: Version{Version: 1, Content: content}}
			err = plan.Revise(incomplete, "motivo", nil, dp)
			if err == nil || !strings.HasPrefix(err.Error(), tt.field+" ") {
				t.Fatalf("revisar: %v", err)
			}
			if plan.Latest.Version != 1 {
				t.Error("cambió la versión al rechazarse")
			}
		})
	}
}

func TestRevise(t *testing.T) {
	stage, task, other := uuid.New(), uuid.New(), uuid.New()
	plan, err := NewPlan(&stage, nil, []uuid.UUID{task}, content, dp)
	if err != nil {
		t.Fatal(err)
	}

	if err := plan.Revise(content, "  ", []uuid.UUID{task}, dp); !errors.Is(err, ErrVersionReasonRequired) {
		t.Fatalf("sin motivo: %v", err)
	}
	if err := plan.Revise(content, "motivo", nil, dp); !errors.Is(err, ErrTasksRequired) {
		t.Fatalf("sin tareas: %v", err)
	}
	if err := plan.Revise(content, " motivo ", []uuid.UUID{other}, dp); err != nil {
		t.Fatal(err)
	}
	if plan.Latest.Version != 2 || plan.Latest.Reason != "motivo" || plan.TaskIDs[0] != other {
		t.Errorf("plan = %+v", plan)
	}
}

func TestReviseBugPlanRejectsTasks(t *testing.T) {
	bug := uuid.New()
	plan, err := NewPlan(nil, &bug, nil, content, dp)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Revise(content, "motivo", []uuid.UUID{uuid.New()}, dp); !errors.Is(err, ErrBugPlanTasks) {
		t.Fatalf("err = %v", err)
	}
}
