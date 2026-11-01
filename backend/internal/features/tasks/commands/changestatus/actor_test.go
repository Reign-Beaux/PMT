package changestatus

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

// actorFor devuelve el actor de las pruebas: el TL de la etapa, que puede liberar,
// cerrar y cancelar.
func actorFor(domain.Status) kernel.Actor {
	return kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
}

// fakeStages responde como una etapa que admite todo, salvo que se indique un error.
type fakeStages struct {
	work   error
	cancel error
}

func (fakeStages) Team(context.Context, uuid.UUID) (kernel.Team, error) {
	return kernel.Team{Lead: "tl", Members: []string{"dev"}}, nil
}

func (s fakeStages) EnsureWorkOpen(context.Context, uuid.UUID) error {
	return s.work
}

func (s fakeStages) EnsureTaskCancellable(context.Context, uuid.UUID) error {
	return s.cancel
}
