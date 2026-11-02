// Package rest proporciona la infraestructura HTTP compartida.
package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

const (
	maxBodyBytes    = 1 << 20
	shutdownTimeout = 10 * time.Second
)

// Serve atiende peticiones hasta que ctx se cancela y luego cierra el servidor.
func Serve(ctx context.Context, addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("servidor http: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("cerrar servidor http: %w", err)
	}
	return nil
}

// DecodeJSON lee el cuerpo JSON de una petición.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return kernel.Invalid("cuerpo inválido: " + err.Error())
	}
	if decoder.More() {
		return kernel.Invalid("cuerpo inválido: contiene más de un valor JSON")
	}
	return nil
}

// PathUUID lee un identificador UUID de la ruta.
func PathUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, kernel.Invalid(name + " inválido")
	}
	return id, nil
}

// WriteJSON escribe una respuesta JSON.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("escribir respuesta: %v", err)
	}
}

// WriteError traduce un error al formato común de la API. Los errores que no
// son de negocio se registran y no exponen su detalle.
func WriteError(w http.ResponseWriter, err error) {
	var businessErr *kernel.Error
	if !errors.As(err, &businessErr) {
		log.Printf("error interno: %v", err)
		writeMessage(w, http.StatusInternalServerError, "error interno")
		return
	}

	status := http.StatusBadRequest
	switch businessErr.Kind {
	case kernel.KindNotFound:
		status = http.StatusNotFound
	case kernel.KindConflict:
		status = http.StatusConflict
	case kernel.KindForbidden:
		status = http.StatusForbidden
	}
	writeMessage(w, status, businessErr.Message)
}

func writeMessage(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}
