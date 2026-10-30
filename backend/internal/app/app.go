// Package app compone la aplicación y gestiona su ciclo de vida.
package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pmt/internal/features/projects"
	"pmt/internal/features/stages"
	"pmt/internal/platform/mcp"
	"pmt/internal/platform/postgres"
	"pmt/internal/platform/rest"
)

const connectTimeout = 5 * time.Second

// Run arranca la aplicación y la mantiene activa hasta recibir una señal de cierre.
func Run() error {
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	pool, err := postgres.Connect(connectCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	mux := http.NewServeMux()
	tools := mcp.NewServer()
	projects.Register(mux, tools, pool)
	stages.Register(mux, tools, pool)
	mux.Handle("/mcp", mcp.Handler(tools))

	return rest.Serve(ctx, cfg.ServerAddr, mux)
}
