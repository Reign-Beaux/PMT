package domain

import "pmt/internal/kernel"

// ErrImportForbidden indica que el perfil no importa planes.
var ErrImportForbidden = kernel.Forbidden("solo PP o el usuario importan proyectos o etapas")

// EnsureCanImport exige que el actor pueda importar. Se comprueba antes de leer el
// fichero o la base.
func EnsureCanImport(actor kernel.Actor) error {
	if !actor.Profile.IsPlanner() {
		return ErrImportForbidden
	}
	return nil
}
