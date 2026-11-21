package changeproductversion

import (
	"net/http"

	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

// asUser hace que la petición actúe como un usuario si la prueba no le dio otro
// actor: toda ruta protegida llega con uno.
func asUser(r *http.Request) *http.Request {
	if rest.ActorFrom(r.Context()) != (kernel.Actor{}) {
		return r
	}
	return r.WithContext(rest.WithActor(r.Context(), kernel.Actor{Name: "saul", Profile: kernel.ProfileUser}))
}
