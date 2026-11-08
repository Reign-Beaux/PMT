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

	"pmt/internal/features/activity"
	"pmt/internal/features/agents"
	"pmt/internal/features/bugs"
	"pmt/internal/features/devplans"
	"pmt/internal/features/imports"
	"pmt/internal/features/projects"
	"pmt/internal/features/qa"
	"pmt/internal/features/reviews"
	"pmt/internal/features/stages"
	"pmt/internal/features/tasks"
	"pmt/internal/features/workflow"
	"pmt/internal/kernel"
	"pmt/internal/platform/events"
	"pmt/internal/platform/mcp"
	"pmt/internal/platform/postgres"
	"pmt/internal/platform/rest"
	"pmt/internal/platform/web"
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
	listener := events.NewListener(events.PgxDialer{DatabaseURL: cfg.DatabaseURL})
	mux.Handle("GET /api/events", events.Handler(listener))
	listenerDone := make(chan struct{})
	go func() {
		listener.Run(ctx)
		close(listenerDone)
	}()

	tools := mcp.NewServer()
	registerFeatures(mux, tools, pool)

	frontend := web.Handler(cfg.FrontendDist)
	serveErr := rest.Serve(ctx, cfg.ServerAddr, routes(mux, tools, agents.NewAuthenticator(pool), frontend))
	stop() // si Serve falló antes de la señal, el listener termina con ctx
	<-listenerDone
	return serveErr
}

// routes protege la API y el MCP con la autenticación por token y sirve el
// frontend compilado en el resto de las rutas.
func routes(api http.Handler, tools *sdk.Server, authenticator kernel.Authenticator, frontend http.Handler) http.Handler {
	root := http.NewServeMux()
	apiHandler := rest.Authenticate(authenticator)(api)
	root.Handle("/api", apiHandler)
	root.Handle("/api/", apiHandler)
	root.Handle("/mcp", mcp.Authenticate(authenticator)(mcp.Handler(tools)))
	root.Handle("/", frontend)
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
	bugChecker := bugs.NewChecker(pool)
	reviews.Register(mux, tools, pool, taskChecker, bugChecker, stageChecker)
	qa.Register(mux, tools, pool, taskChecker, stageChecker)
	devplans.Register(mux, tools, pool, taskChecker, stageChecker, bugChecker)
	activity.Register(mux, tools, pool, projectChecker, stageChecker, taskChecker, bugChecker)
}
