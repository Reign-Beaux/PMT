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

	"pmt/internal/features/agents"
	"pmt/internal/features/bugs"
	"pmt/internal/features/imports"
	"pmt/internal/features/projects"
	"pmt/internal/features/qa"
	"pmt/internal/features/reviews"
	"pmt/internal/features/stages"
	"pmt/internal/features/tasks"
	"pmt/internal/features/workflow"
	"pmt/internal/kernel"
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

	return rest.Serve(ctx, cfg.ServerAddr, routes(mux, tools, agents.NewAuthenticator(pool)))
}

// routes protege la API y el MCP con la autenticación por token.
func routes(api http.Handler, tools *sdk.Server, authenticator kernel.Authenticator) http.Handler {
	root := http.NewServeMux()
	root.Handle("/api/", rest.Authenticate(authenticator)(api))
	root.Handle("/mcp", mcp.Authenticate(authenticator)(mcp.Handler(tools)))
	return root
}

// registerFeatures registra las rutas REST y las herramientas MCP de cada feature.
func registerFeatures(mux *http.ServeMux, tools *sdk.Server, pool *pgxpool.Pool) {
	projectChecker := projects.NewChecker(pool)
	agentChecker := agents.NewChecker(pool)

	projects.Register(mux, tools, pool, stages.NewProgress(pool))
	taskChecker := tasks.NewChecker(pool)
	stages.Register(mux, tools, pool, taskChecker, projectChecker, agentChecker, workflow.NewCloser(pool))
	stageChecker := stages.NewChecker(pool, projectChecker)
	tasks.Register(mux, tools, pool, stageChecker, agentChecker)
	imports.Register(tools, pool)
	workflow.Register(mux, tools, pool, taskChecker)
	bugs.Register(mux, tools, pool, projectChecker)
	reviews.Register(mux, tools, pool, taskChecker, bugs.NewChecker(pool), stageChecker)
	qa.Register(mux, tools, pool, taskChecker, stageChecker)
}
