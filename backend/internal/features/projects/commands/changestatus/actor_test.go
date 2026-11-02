package changestatus

import (
	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
)

// actorFor devuelve el actor de las pruebas: el usuario puede todo en un proyecto.
func actorFor(domain.Status) kernel.Actor {
	return kernel.Owner
}
