package changestatus

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/kernel"
)

// actorFor devuelve el actor de las pruebas: QA confirma y rechaza lo reportado.
func actorFor(domain.Status) kernel.Actor {
	return kernel.Actor{Name: "qa", Profile: kernel.ProfileQA}
}

// fakeProjects responde como un proyecto en working, salvo que se indique un error.
type fakeProjects struct {
	err error
}

func (p fakeProjects) EnsureWorking(context.Context, uuid.UUID) error {
	return p.err
}
