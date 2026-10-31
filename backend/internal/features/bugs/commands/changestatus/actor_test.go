package changestatus

import (
	"pmt/internal/features/bugs/domain"
	"pmt/internal/kernel"
)

// actorFor devuelve el actor de las pruebas: QA confirma y rechaza lo reportado.
func actorFor(domain.Status) kernel.Actor {
	return kernel.Actor{Name: "qa", Profile: kernel.ProfileQA}
}
