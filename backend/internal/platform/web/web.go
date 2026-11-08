// Package web sirve el frontend compilado con fallback SPA.
package web

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

// Handler devuelve un handler que sirve el directorio dist con fallback SPA a
// index.html. Ninguna ruta escapa de dist: los .. y los enlaces simbólicos
// externos quedan confinados por os.OpenRoot. Si dist no existe o no tiene
// index.html, registra un aviso y devuelve un handler que responde 404.
func Handler(dist string) http.Handler {
	root, err := os.OpenRoot(dist)
	if err != nil {
		return sinFrontend(dist, err)
	}
	fsys := root.FS()
	if _, err := fs.Stat(fsys, "index.html"); err != nil {
		return sinFrontend(dist, err)
	}

	fileServer := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
			return
		}
		// La ruta se limpia antes de decidir: los .., %2e%2e y barras dobles
		// quedan confinados en dist y las rutas resultantes inexistentes caen
		// al fallback SPA, como en un servidor http real.
		cleaned := path.Clean("/" + r.URL.Path)
		if cleaned != r.URL.Path {
			rc := r.Clone(r.Context())
			rc.URL.Path = cleaned
			r = rc
		}
		name := strings.TrimPrefix(cleaned, "/")
		if name != "" {
			if info, err := fs.Stat(fsys, name); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFileFS(w, r, fsys, "index.html")
	})
}

func sinFrontend(dist string, err error) http.Handler {
	log.Printf("web: frontend no compilado en %s: %v", dist, err)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "frontend no compilado", http.StatusNotFound)
	})
}
