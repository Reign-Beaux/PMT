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

type fakeStages struct{}

func (fakeStages) Team(context.Context, uuid.UUID) (kernel.Team, error) {
	return kernel.Team{Lead: "tl", Members: []string{"dev"}}, nil
}
