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
		return fmt.Errorf("cuerpo inválido: %w", err)
	}
	if decoder.More() {
		return errors.New("cuerpo inválido: contiene más de un valor JSON")
	}
	return nil
}

// WriteJSON escribe una respuesta JSON.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("escribir respuesta: %v", err)
	}
}

// WriteError escribe un error con el formato común de la API.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}
