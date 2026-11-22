package events

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

// membresias simula la membresía: el usuario ve los proyectos del mapa y cuenta
// cuántas veces se consultó.
type membresias struct {
	ve        map[uuid.UUID]bool
	consultas int
	err       error
}

func (m *membresias) visible(_ context.Context, _ kernel.Actor, projectID uuid.UUID) (bool, error) {
	m.consultas++
	return m.ve[projectID], m.err
}

func TestFiltroEntregaSoloLosProyectosQueVe(t *testing.T) {
	suyo, ajeno := uuid.New(), uuid.New()
	ana := kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: uuid.New()}
	m := &membresias{ve: map[uuid.UUID]bool{suyo: true}}
	f := newFilter(ana, m.visible)
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		event ChangeEvent
		want  bool
	}{
		{"resync", ResyncEvent(), true},
		{"tarea de su proyecto", ChangeEvent{Type: TypeInvalidate, Resource: ResourceTask, ProjectID: suyo}, true},
		{"tarea de un proyecto ajeno", ChangeEvent{Type: TypeInvalidate, Resource: ResourceTask, ProjectID: ajeno}, false},
		{"sin proyecto", ChangeEvent{Type: TypeInvalidate, Resource: ResourceStage}, false},
		{"la quitaron de un proyecto ajeno", ChangeEvent{Type: TypeInvalidate, Resource: ResourceProject, ProjectID: ajeno, UserID: ana.UserID}, true},
		{"cambio de miembros de otro usuario", ChangeEvent{Type: TypeInvalidate, Resource: ResourceProject, ProjectID: ajeno, UserID: uuid.New()}, false},
	} {
		if got := f.allows(ctx, tc.event); got != tc.want {
			t.Errorf("%s: entrega = %v", tc.name, got)
		}
	}
}

func TestFiltroRecuerdaHastaUnCambioDeProyecto(t *testing.T) {
	project := uuid.New()
	m := &membresias{ve: map[uuid.UUID]bool{}}
	f := newFilter(kernel.Actor{UserID: uuid.New()}, m.visible)
	ctx := context.Background()
	task := ChangeEvent{Type: TypeInvalidate, Resource: ResourceTask, ProjectID: project}

	f.allows(ctx, task)
	f.allows(ctx, task)
	if m.consultas != 1 {
		t.Fatalf("consultas = %d, quiero 1", m.consultas)
	}

	// Lo comparten: el cambio de miembros llega como cambio de proyecto.
	m.ve[project] = true
	if !f.allows(ctx, ChangeEvent{Type: TypeInvalidate, Resource: ResourceProject, ProjectID: project}) {
		t.Fatal("debe recibir el cambio del proyecto que ahora ve")
	}
	if !f.allows(ctx, task) || m.consultas != 2 {
		t.Fatalf("tras compartir: consultas = %d", m.consultas)
	}
}

func TestFiltroNoEntregaSiFallaLaMembresia(t *testing.T) {
	m := &membresias{err: errors.New("sin base")}
	f := newFilter(kernel.Actor{UserID: uuid.New()}, m.visible)

	if f.allows(context.Background(), ChangeEvent{Type: TypeInvalidate, Resource: ResourceTask, ProjectID: uuid.New()}) {
		t.Fatal("no debe entregar si no sabe si lo ve")
	}
}
