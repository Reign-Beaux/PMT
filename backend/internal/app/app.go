// Package app compone la aplicación y gestiona su ciclo de vida.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/activity"
	"pmt/internal/features/agents"
	"pmt/internal/features/auth"
	"pmt/internal/features/auth/commands/seedowner"
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
	"pmt/internal/platform/config"
	"pmt/internal/platform/events"
	"pmt/internal/platform/health"
	"pmt/internal/platform/mcp"
	"pmt/internal/platform/postgres"
	"pmt/internal/platform/rest"
	"pmt/internal/platform/web"
)

const connectTimeout = 5 * time.Second

// Run aplica las migraciones pendientes, arranca la aplicación y la mantiene
// activa hasta recibir una señal de cierre.
func Run(cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	pool, err := postgres.Connect(connectCtx, cfg.Database.ConnectionString)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err = postgres.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("aplicar migraciones: %w", err)
	}

	mux := http.NewServeMux()
	members := guard{members: projects.NewMembership(pool)}
	listener := events.NewListener(events.PgxDialer{DatabaseURL: cfg.Database.ConnectionString})
	mux.Handle("GET /api/events", events.Handler(listener, members.visible))
	listenerDone := make(chan struct{})
	go func() {
		listener.Run(ctx)
		close(listenerDone)
	}()

	users := auth.New(pool, agents.NewChecker(pool), auth.Settings{
		PublicURL:    cfg.App.PublicURL,
		CookieSecure: cfg.Auth.CookieSecure,
	})
	if err = users.SeedOwner(ctx, seedowner.Command{
		Email: cfg.Owner.Email, Password: cfg.Owner.Password, Username: cfg.Owner.Username,
	}); err != nil {
		return fmt.Errorf("crear dueño: %w", err)
	}
	if err = adoptOrphans(ctx, users, pool); err != nil {
		return err
	}
	users.Register(mux)

	tools := mcp.NewServer()
	registerFeatures(mux, tools, pool)
	api := protect(mux, tools, members)

	frontend := web.Handler(cfg.Frontend.Dist)
	authenticator := credentials{users: users, agents: agents.NewAuthenticator(pool), owner: users}
	serveErr := rest.Serve(ctx, cfg.Server.Addr, routes(api, tools, authenticator, pool, frontend))
	stop() // si Serve falló antes de la señal, el listener termina con ctx
	<-listenerDone
	return serveErr
}

// adoptOrphans deja al dueño de la instancia como owner de los proyectos sin
// owner: los que existían cuando la migración de miembros no encontró usuarios.
func adoptOrphans(ctx context.Context, users *auth.Feature, pool *pgxpool.Pool) error {
	owner, err := users.InstanceOwnerID(ctx)
	if err != nil {
		return fmt.Errorf("buscar dueño de la instancia: %w", err)
	}
	adopted, err := projects.AdoptOrphans(ctx, pool, owner)
	if err != nil {
		return err
	}
	if adopted > 0 {
		slog.Info("el dueño de la instancia quedó como owner de los proyectos sin owner", "proyectos", adopted)
	}
	return nil
}

// credentials resuelve las credenciales de una petición: la cookie es siempre de
// un usuario y actúa como User; un token con prefijo pmt_ es de usuario y actúa
// con el perfil que declare la sesión; cualquier otro es de agente, conserva su
// perfil fijo y actúa con la membresía del dueño de la instancia.
type credentials struct {
	users interface {
		AuthenticateSession(ctx context.Context, session string) (kernel.Actor, error)
		AuthenticateToken(ctx context.Context, token string, declared kernel.Profile) (kernel.Actor, error)
	}
	agents interface {
		Authenticate(ctx context.Context, token string) (kernel.Actor, error)
	}
	owner interface {
		InstanceOwnerID(ctx context.Context) (uuid.UUID, error)
	}
}

func (c credentials) AuthenticateSession(ctx context.Context, session string) (kernel.Actor, error) {
	return c.users.AuthenticateSession(ctx, session)
}

func (c credentials) AuthenticateToken(ctx context.Context, token string, declared kernel.Profile) (kernel.Actor, error) {
	if auth.IsUserToken(token) {
		return c.users.AuthenticateToken(ctx, token, declared)
	}
	agent, err := c.agents.Authenticate(ctx, token)
	if err != nil {
		return kernel.Actor{}, err
	}
	actor, err := kernel.AsAgent(agent, declared)
	if err != nil {
		return kernel.Actor{}, err
	}
	if actor.UserID, err = c.owner.InstanceOwnerID(ctx); err != nil {
		return kernel.Actor{}, err
	}
	return actor, nil
}

// routes protege la API y el MCP con la autenticación, expone las sondas de
// salud y las rutas públicas de auth sin credenciales y sirve el frontend
// compilado en el resto de las rutas.
func routes(api http.Handler, tools *sdk.Server, authenticator kernel.Authenticator, pinger health.Pinger, frontend http.Handler) http.Handler {
	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", health.Live)
	root.HandleFunc("GET /readyz", health.Ready(pinger))
	apiHandler := rest.Authenticate(authenticator, auth.IsPublic)(api)
	root.Handle("/api", apiHandler)
	root.Handle("/api/", apiHandler)
	root.Handle("/mcp", mcp.Authenticate(authenticator)(mcp.Handler(tools)))
	root.Handle("/", frontend)
	return root
}

// protect hace que toda herramienta y toda ruta de la API compruebe la membresía
// antes de llegar a su handler (membership.go). Devuelve la API protegida.
func protect(mux *http.ServeMux, tools *sdk.Server, members guard) http.Handler {
	mcp.Guard(tools, members.tool)
	return rest.Guard(mux, members.route)(mux)
}

// registerFeatures registra las rutas REST y las herramientas MCP de cada feature.
func registerFeatures(mux *http.ServeMux, tools *sdk.Server, pool *pgxpool.Pool) {
	projectChecker := projects.NewChecker(pool)
	agentChecker := agents.NewChecker(pool)
	userChecker := auth.NewChecker(pool)

	projects.Register(mux, tools, pool, stages.NewProgress(pool), userChecker)
	taskChecker := tasks.NewChecker(pool)
	workers := projects.NewWorkers(pool, userChecker)
	stages.Register(mux, tools, pool, taskChecker, projectChecker, agentChecker, workers, workflow.NewCloser(pool))
	stageChecker := stages.NewChecker(pool, projectChecker)
	tasks.Register(mux, tools, pool, stageChecker, agentChecker, workers)
	imports.Register(tools, pool)
	workflow.Register(mux, tools, pool, taskChecker)
	bugs.Register(mux, tools, pool, projectChecker)
	bugChecker := bugs.NewChecker(pool)
	reviews.Register(mux, tools, pool, taskChecker, bugChecker, stageChecker)
	qa.Register(mux, tools, pool, taskChecker, stageChecker)
	devplans.Register(mux, tools, pool, taskChecker, stageChecker, bugChecker)
	activity.Register(mux, tools, pool, projectChecker, stageChecker, taskChecker, bugChecker)
}
