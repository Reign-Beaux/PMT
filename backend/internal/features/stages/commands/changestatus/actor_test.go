package changestatus

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// actorFor devuelve un actor que puede hacer la transición: el TL de la etapa si
// va a in_dev o done, y el usuario en lo demás.
func actorFor(status domain.Status) kernel.Actor {
	if status == domain.StatusInDev || status == domain.StatusDone {
		return kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	}
	return kernel.Owner
}

// fakeProjects responde como un proyecto con esos permisos.
type fakeProjects struct {
	accepts bool
	frozen  bool
	err     error
}

func (p fakeProjects) AcceptsWork(context.Context, uuid.UUID) (bool, error) {
	return p.accepts, p.err
}

func (p fakeProjects) IsFrozen(context.Context, uuid.UUID) (bool, error) {
	return p.frozen, p.err
}

var (
	// workingProject admite cualquier transición.
	workingProject = fakeProjects{accepts: true}
	// waitingProject está en planning o stand_by.
	waitingProject = fakeProjects{}
	// frozenProject está detenido o cancelado.
	frozenProject = fakeProjects{frozen: true}
)
