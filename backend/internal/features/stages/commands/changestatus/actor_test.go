package changestatus

import (
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
