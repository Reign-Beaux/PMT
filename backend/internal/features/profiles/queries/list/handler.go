// Package list consulta los perfiles que PMT sirve.
package list

import "pmt/internal/features/profiles/catalog"

// Handler ejecuta la operación.
type Handler struct{}

// New construye la operación.
func New() *Handler { return &Handler{} }

// Handle devuelve los perfiles: nombre de sesión, perfil y descripción corta.
func (h *Handler) Handle() []catalog.Entry {
	return catalog.List()
}
