// Package downloads sirve el CLI pmt: /install.sh con la URL pública del
// servidor y los binarios en /descargas. Ambas rutas son públicas.
package downloads

import (
	_ "embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
)

//go:embed install.sh
var installScript string

// Sums es el archivo con las sumas SHA-256 de los binarios.
const Sums = "SHA256SUMS"

// Binaries son los binarios que se publican, uno por sistema y arquitectura.
var Binaries = []string{"pmt-linux-amd64", "pmt-linux-arm64", "pmt-darwin-amd64", "pmt-darwin-arm64"}

// sinBinarios es el script que responde /install.sh cuando el servidor no tiene
// los binarios: con curl -fsSL | sh, la persona lee el motivo.
const sinBinarios = "#!/bin/sh\necho \"pmt: este servidor no tiene los binarios del CLI; avisa a quien lo administra.\" >&2\nexit 1\n"

// Mount agrega a next las rutas GET /install.sh y GET /descargas/{archivo}.
// dist es cli.dist. Si está vacío, no existe o le falta SHA256SUMS, el servidor
// sigue: /install.sh responde un script que lo explica y /descargas, 404. La
// carpeta se revisa en cada petición, así que basta compilar los binarios.
func Mount(dist, publicURL string, next http.Handler) http.Handler {
	h := handler{dist: dist, script: strings.ReplaceAll(installScript, "__PMT_SERVER__", strings.TrimRight(publicURL, "/"))}
	if !h.available() {
		log.Printf("downloads: no hay binarios del CLI en %q; /install.sh avisará que faltan", dist)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /install.sh", h.install)
	mux.HandleFunc("GET /descargas/{file}", h.file)
	mux.Handle("/", next)
	return mux
}

type handler struct {
	dist   string
	script string
}

// open abre dist confinado: ninguna ruta escapa de la carpeta. Quien la abre la cierra.
func (h handler) open() (*os.Root, bool) {
	if h.dist == "" {
		return nil, false
	}
	root, err := os.OpenRoot(h.dist)
	return root, err == nil
}

func (h handler) available() bool {
	root, ok := h.open()
	if !ok {
		return false
	}
	defer root.Close()
	_, err := fs.Stat(root.FS(), Sums)
	return err == nil
}

func (h handler) install(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if !h.available() {
		_, _ = w.Write([]byte(sinBinarios))
		return
	}
	_, _ = w.Write([]byte(h.script))
}

func (h handler) file(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("file")
	if name != Sums && !slices.Contains(Binaries, name) {
		http.NotFound(w, r)
		return
	}
	root, ok := h.open()
	if !ok {
		http.Error(w, "este servidor no tiene los binarios del CLI", http.StatusNotFound)
		return
	}
	defer root.Close()
	fsys := root.FS()
	if _, err := fs.Stat(fsys, name); err != nil {
		http.Error(w, "este servidor no tiene los binarios del CLI", http.StatusNotFound)
		return
	}
	if name == Sums {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFileFS(w, r, fsys, name)
}
