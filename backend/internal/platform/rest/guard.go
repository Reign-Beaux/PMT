package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// Router da la ruta que atiende una petición, como http.ServeMux.
type Router interface {
	Handler(r *http.Request) (http.Handler, string)
}

// RouteCheck revisa una petición antes de que llegue a su handler. pattern es la
// ruta registrada que la atiende ("GET /api/stages/{id}"), y la petición ya trae
// sus valores de ruta (r.PathValue). Un error la rechaza con WriteError.
type RouteCheck func(r *http.Request, pattern string) error

// Guard hace que toda petición que atienda router pase antes por check. Las que
// no atiende ninguna ruta siguen sin revisar, para que respondan 404 o 405.
// Va dentro de Authenticate, para que la petición ya traiga su actor.
func Guard(router Router, check RouteCheck) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, pattern := router.Handler(r)
			if pattern == "" {
				next.ServeHTTP(w, r)
				return
			}
			setPathValues(r, pattern)
			if err := check(r, pattern); err != nil {
				WriteError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// setPathValues llena los valores de ruta de la petición según su patrón. El
// mux los vuelve a llenar al despachar; aquí sirven para revisarla antes. Las
// rutas de PMT solo tienen comodines de un segmento ({id}).
func setPathValues(r *http.Request, pattern string) {
	_, path, found := strings.Cut(pattern, " ")
	if !found {
		path = pattern
	}
	names := strings.Split(strings.Trim(path, "/"), "/")
	values := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(names) != len(values) {
		return
	}
	for i, name := range names {
		if strings.HasPrefix(name, "{") && strings.HasSuffix(name, "}") {
			r.SetPathValue(strings.TrimSuffix(strings.TrimPrefix(name, "{"), "}"), values[i])
		}
	}
}

// PeekJSON lee el cuerpo JSON de la petición sin consumirlo: lo deja para el
// handler. Devuelve nil si no hay cuerpo, si no es un objeto JSON o si pasa del
// límite; el handler lo rechazará al decodificarlo.
func PeekJSON(r *http.Request) map[string]any {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	rest := r.Body
	r.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(raw), rest), rest}
	if err != nil || len(raw) > maxBodyBytes {
		return nil
	}
	var body map[string]any
	if json.Unmarshal(raw, &body) != nil {
		return nil
	}
	return body
}
