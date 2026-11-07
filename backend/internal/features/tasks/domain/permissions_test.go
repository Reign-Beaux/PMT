package domain

import (
	"errors"
	"testing"

	"pmt/internal/kernel"
)

// allowedChange devuelve una tarea, un actor y un equipo con los que la transición
// pasa perfil y ámbito. Sin perfiles devuelve un actor cualquiera.
func allowedChange(from, to Status) (Task, kernel.Actor, kernel.Team) {
	actor := kernel.Owner
	if profiles := statusProfiles[from][to]; len(profiles) > 0 {
		actor = kernel.Actor{Name: "agente", Profile: profiles[0]}
	}
	return Task{Status: from, Assignee: actor.Name}, actor, kernel.Team{Lead: actor.Name, Members: []string{actor.Name}}
}

func TestEveryTransitionHasProfiles(t *testing.T) {
	for from, targets := range statusTransitions {
		for _, to := range targets {
			if len(statusProfiles[from][to]) == 0 {
				t.Errorf("%s a %s no tiene perfiles", from, to)
			}
		}
		if len(statusProfiles[from]) != len(targets) {
			t.Errorf("%s tiene perfiles para transiciones que no existen", from)
		}
	}
}

func TestChangeStatusValidatesProfile(t *testing.T) {
	team := kernel.Team{Lead: "tl", Members: []string{"dev"}}
	tl := kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	otherTL := kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}
	dev := kernel.Actor{Name: "dev", Profile: kernel.ProfileDevBackend}
	front := kernel.Actor{Name: "dev", Profile: kernel.ProfileDevFrontend}
	otherDev := kernel.Actor{Name: "dev2", Profile: kernel.ProfileDevBackend}
	cr := kernel.Actor{Name: "cr", Profile: kernel.ProfileCR}
	qa := kernel.Actor{Name: "qa", Profile: kernel.ProfileQA}
	pp := kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}
	pc := kernel.Actor{Name: "pc", Profile: kernel.ProfilePC}

	tests := []struct {
		name   string
		task   Task
		to     Status
		actor  kernel.Actor
		reason string
		want   error
	}{
		{"CR pasa a cierre con motivo", Task{Status: StatusInReview}, StatusPendingClose, cr, "interno", nil},
		{"CR a cierre sin motivo", Task{Status: StatusInReview}, StatusPendingClose, cr, "  ", kernel.ErrReasonRequired},
		{"TL devuelve a CR", Task{Status: StatusPendingClose}, StatusInReview, tl, "falta", nil},
		{"TL devuelve a QA", Task{Status: StatusPendingClose}, StatusInQA, tl, "falta", nil},
		{"devolver sin motivo", Task{Status: StatusPendingClose}, StatusInQA, tl, "", kernel.ErrReasonRequired},
		{"otro TL no devuelve", Task{Status: StatusPendingClose}, StatusInQA, otherTL, "falta", kernel.ErrNotStageLead},
		{"QA no devuelve por TL", Task{Status: StatusPendingClose}, StatusInReview, qa, "falta", kernel.ErrTransitionForbidden},
		{"sin devolución a correcciones", Task{Status: StatusPendingClose}, StatusChangesRequested, tl, "falta", ErrTaskStatusTransition},
		{"corrección vuelve a in_dev", Task{Status: StatusChangesRequested, Assignee: "dev"}, StatusInDev, dev, "", nil},
		{"sin atajo a in_review", Task{Status: StatusChangesRequested, Assignee: "dev"}, StatusInReview, dev, "", ErrTaskStatusTransition},
		{"TL libera", Task{Status: StatusBacklog}, StatusReady, tl, "", nil},
		{"TL ajeno no libera", Task{Status: StatusBacklog}, StatusReady, otherTL, "", kernel.ErrNotStageLead},
		{"PP no libera", Task{Status: StatusBacklog}, StatusReady, pp, "", kernel.ErrTransitionForbidden},
		{"dev toma", Task{Status: StatusReady}, StatusInDev, dev, "", nil},
		{"frontend toma", Task{Status: StatusReady}, StatusInDev, front, "", nil},
		{"dev ajeno no toma", Task{Status: StatusReady}, StatusInDev, otherDev, "", ErrNotStageMember},
		{"CR no toma", Task{Status: StatusReady}, StatusInDev, cr, "", kernel.ErrTransitionForbidden},
		{"dueño entrega", Task{Status: StatusInDev, Assignee: "dev"}, StatusInReview, dev, "", nil},
		{"otro dev no entrega", Task{Status: StatusInDev, Assignee: "dev"}, StatusInReview, otherDev, "", ErrNotAssignee},
		{"sin dueño no se entrega", Task{Status: StatusInDev}, StatusInReview, dev, "", ErrNotAssignee},
		{"CR aprueba", Task{Status: StatusInReview}, StatusInQA, cr, "", nil},
		{"QA no hace de CR", Task{Status: StatusInReview}, StatusInQA, qa, "", kernel.ErrTransitionForbidden},
		{"QA aprueba", Task{Status: StatusInQA}, StatusPendingClose, qa, "verificado en /tmp/qa.log", nil},
		{"aprobar sin motivo", Task{Status: StatusInQA}, StatusPendingClose, qa, "", kernel.ErrReasonRequired},
		{"aprobar con solo espacios", Task{Status: StatusInQA}, StatusPendingClose, qa, "  ", kernel.ErrReasonRequired},
		{"usuario no aprueba", Task{Status: StatusInQA}, StatusPendingClose, kernel.Owner, "verificado", kernel.ErrTransitionForbidden},
		{"TL cierra", Task{Status: StatusPendingClose}, StatusDone, tl, "", nil},
		{"usuario no cierra", Task{Status: StatusPendingClose}, StatusDone, kernel.Owner, "", kernel.ErrTransitionForbidden},
		{"PP cancela con motivo", Task{Status: StatusReady}, StatusCancelled, pp, "ya no", nil},
		{"usuario cancela con motivo", Task{Status: StatusReady}, StatusCancelled, kernel.Owner, "ya no", nil},
		{"cancelar sin motivo", Task{Status: StatusReady}, StatusCancelled, pp, "  ", kernel.ErrReasonRequired},
		{"TL ajeno no cancela", Task{Status: StatusReady}, StatusCancelled, otherTL, "ya no", kernel.ErrNotStageLead},
		{"reabrir sin motivo", Task{Status: StatusCancelled}, StatusBacklog, pp, "", kernel.ErrReasonRequired},
		{"PC no mueve tareas", Task{Status: StatusReady}, StatusCancelled, pc, "ya no", kernel.ErrTransitionForbidden},
		{"perfil antes que motivo", Task{Status: StatusReady}, StatusCancelled, cr, "", kernel.ErrTransitionForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, from := tt.task, tt.task.Status
			err := task.ChangeStatus(tt.to, tt.actor, tt.reason, team)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if tt.want == nil && task.Status != tt.to {
				t.Fatalf("status = %s", task.Status)
			}
			if tt.want != nil && task.Status != from {
				t.Fatalf("el status cambió a %s", task.Status)
			}
		})
	}
}

func TestChangeStatusTracksAssignee(t *testing.T) {
	dev := kernel.Actor{Name: "dev", Profile: kernel.ProfileDevBackend}
	task := Task{Status: StatusReady}
	if err := task.ChangeStatus(StatusInDev, dev, "", kernel.Team{Members: []string{"dev"}}); err != nil {
		t.Fatal(err)
	}
	if task.Assignee != "dev" {
		t.Fatalf("assignee = %q", task.Assignee)
	}

	task.Status = StatusCancelled
	if err := task.ChangeStatus(StatusBacklog, kernel.Owner, "vuelve", kernel.Team{}); err != nil {
		t.Fatal(err)
	}
	if task.Assignee != "" {
		t.Fatalf("assignee = %q, debía limpiarse", task.Assignee)
	}
}

func TestAssign(t *testing.T) {
	for _, status := range []Status{StatusInDev, StatusInReview, StatusChangesRequested, StatusInQA, StatusPendingClose} {
		task := Task{Status: status}
		if err := task.Assign("dev"); err != nil || task.Assignee != "dev" {
			t.Fatalf("%s: err = %v, assignee = %q", status, err, task.Assignee)
		}
	}
	for _, status := range []Status{StatusBacklog, StatusReady, StatusDone, StatusCancelled} {
		task := Task{Status: status}
		if err := task.Assign("dev"); !errors.Is(err, ErrTaskNotTaken) {
			t.Fatalf("%s: err = %v", status, err)
		}
	}
}

func TestEnsureCanPlanOnlyPlanners(t *testing.T) {
	for _, p := range []kernel.Profile{kernel.ProfilePP, kernel.ProfilePC, kernel.ProfileTL, kernel.ProfileDevFrontend, kernel.ProfileDevBackend, kernel.ProfileCR, kernel.ProfileQAP, kernel.ProfileQA, kernel.ProfileDP, kernel.ProfileUser} {
		err := EnsureCanPlan(kernel.Actor{Name: "agente", Profile: p})
		if p == kernel.ProfilePP || p == kernel.ProfileUser {
			if err != nil {
				t.Errorf("%s: err = %v", p, err)
			}
			continue
		}
		if !errors.Is(err, ErrPlanForbidden) {
			t.Errorf("%s: err = %v", p, err)
		}
	}
}

func TestEnsureCanChangeStatusRejectsPC(t *testing.T) {
	if err := EnsureCanChangeStatus(kernel.Actor{Name: "pc", Profile: kernel.ProfilePC}); !errors.Is(err, kernel.ErrTransitionForbidden) {
		t.Errorf("PC: err = %v", err)
	}
	for _, targets := range statusProfiles {
		for _, profiles := range targets {
			for _, p := range profiles {
				if err := EnsureCanChangeStatus(kernel.Actor{Name: "agente", Profile: p}); err != nil {
					t.Errorf("%s: err = %v", p, err)
				}
			}
		}
	}
}
