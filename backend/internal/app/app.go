// Package app compone la aplicación y gestiona su ciclo de vida.
package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/bugs"
	"pmt/internal/features/imports"
	"pmt/internal/features/projects"
	"pmt/internal/features/stages"
	"pmt/internal/features/tasks"
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
	registerFeatures(mux, tools, pool)
	mux.Handle("/mcp", mcp.Handler(tools))

	return rest.Serve(ctx, cfg.ServerAddr, mux)
}

// registerFeatures registra las rutas REST y las herramientas MCP de cada feature.
func registerFeatures(mux *http.ServeMux, tools *sdk.Server, pool *pgxpool.Pool) {
	projectChecker := projects.NewChecker(pool)

	projects.Register(mux, tools, pool)
	stages.Register(mux, tools, pool, tasks.NewChecker(pool), projectChecker)
	tasks.Register(mux, tools, pool, stages.NewChecker(pool, projectChecker))
	imports.Register(tools, pool)
	bugs.Register(mux, tools, pool, projectChecker)
}
