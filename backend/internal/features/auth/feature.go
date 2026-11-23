// Package auth construye las slices de usuarios, sesiones, invitaciones y tokens
// de API. Solo tiene REST: el MCP no administra cuentas.
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"pmt/internal/features/auth/commands/acceptinvitation"
	"pmt/internal/features/auth/commands/authorizeclilogin"
	"pmt/internal/features/auth/commands/cancelclilogin"
	"pmt/internal/features/auth/commands/changepassword"
	"pmt/internal/features/auth/commands/createinvitation"
	"pmt/internal/features/auth/commands/createtoken"
	"pmt/internal/features/auth/commands/disableuser"
	"pmt/internal/features/auth/commands/enableuser"
	"pmt/internal/features/auth/commands/login"
	"pmt/internal/features/auth/commands/logout"
	"pmt/internal/features/auth/commands/pollclilogin"
	"pmt/internal/features/auth/commands/revoketoken"
	"pmt/internal/features/auth/commands/seedowner"
	"pmt/internal/features/auth/commands/startclilogin"
	"pmt/internal/features/auth/domain"
	"pmt/internal/features/auth/queries/authenticate"
	"pmt/internal/features/auth/queries/getclilogin"
	"pmt/internal/features/auth/queries/getinvitation"
	"pmt/internal/features/auth/queries/instanceowner"
	"pmt/internal/features/auth/queries/listinvitations"
	"pmt/internal/features/auth/queries/listtokens"
	"pmt/internal/features/auth/queries/listusers"
	"pmt/internal/features/auth/queries/me"
	"pmt/internal/features/auth/store"
	"pmt/internal/kernel"
)

// Settings es la configuración que usa la feature.
type Settings struct {
	// PublicURL es la base de los enlaces de invitación (app.public_url).
	PublicURL string
	// CookieSecure marca la cookie de sesión como Secure (auth.cookie_secure).
	CookieSecure bool
}

// Repository es la persistencia de todas las slices. store.Users la cumple.
type Repository interface {
	login.Repository
	logout.Repository
	seedowner.Repository
	createinvitation.Repository
	acceptinvitation.Repository
	createtoken.Repository
	revoketoken.Repository
	changepassword.Repository
	disableuser.Repository
	enableuser.Repository
	startclilogin.Repository
	pollclilogin.Repository
	authorizeclilogin.Repository
	cancelclilogin.Repository
	authenticate.Reader
	getinvitation.Reader
	listtokens.Reader
	listinvitations.Reader
	listusers.Reader
	instanceowner.Reader
	getclilogin.Reader
}

// Feature reúne las slices de la feature.
type Feature struct {
	repo     Repository
	agents   domain.AgentNames
	settings Settings
	auth     *authenticate.Handler
}

// New crea la feature. agents es la feature agents: un username no puede ser el
// nombre de un agente.
func New(db store.DB, agents domain.AgentNames, settings Settings) *Feature {
	return NewWithRepository(store.NewUsers(db), agents, settings)
}

// NewWithRepository crea la feature sobre otra persistencia, como la de pruebas.
func NewWithRepository(repo Repository, agents domain.AgentNames, settings Settings) *Feature {
	return &Feature{repo: repo, agents: agents, settings: settings, auth: authenticate.New(repo)}
}

// Register registra las rutas REST.
func (f *Feature) Register(mux *http.ServeMux) {
	secure := f.settings.CookieSecure
	login.New(f.repo).Register(mux, secure)
	logout.New(f.repo).Register(mux, secure)
	me.New(f.repo).Register(mux)
	changepassword.New(f.repo).Register(mux)
	createtoken.New(f.repo).Register(mux)
	listtokens.New(f.repo).Register(mux)
	revoketoken.New(f.repo).Register(mux)
	createinvitation.New(f.repo, f.settings.PublicURL).Register(mux)
	listinvitations.New(f.repo).Register(mux)
	getinvitation.New(f.repo).Register(mux)
	acceptinvitation.New(f.repo, f.agents).Register(mux, secure)
	listusers.New(f.repo).Register(mux)
	disableuser.New(f.repo).Register(mux)
	enableuser.New(f.repo).Register(mux)
	startclilogin.New(f.repo, f.settings.PublicURL).Register(mux)
	pollclilogin.New(f.repo).Register(mux)
	getclilogin.New(f.repo).Register(mux)
	authorizeclilogin.New(f.repo).Register(mux)
	cancelclilogin.New(f.repo).Register(mux)
}

// SeedOwner crea al dueño de la instancia si no hay usuarios.
func (f *Feature) SeedOwner(ctx context.Context, cmd seedowner.Command) error {
	return seedowner.New(f.repo, f.agents).Handle(ctx, cmd)
}

// AuthenticateSession resuelve la cookie de sesión a su usuario.
func (f *Feature) AuthenticateSession(ctx context.Context, session string) (kernel.Actor, error) {
	return f.auth.Session(ctx, session)
}

// AuthenticateToken resuelve un token de API de usuario al actor con el perfil
// que declara la sesión, o User si no declara ninguno.
func (f *Feature) AuthenticateToken(ctx context.Context, token string, declared kernel.Profile) (kernel.Actor, error) {
	actor, err := f.auth.Token(ctx, token)
	if err != nil {
		return kernel.Actor{}, err
	}
	declaredActor := kernel.AsUser(actor.Name, declared)
	declaredActor.UserID = actor.UserID
	declaredActor.UserName = actor.UserName
	return declaredActor, nil
}

// InstanceOwnerID devuelve el ID del dueño de la instancia, o uuid.Nil si
// todavía no existe. Los tokens de agente actúan con su membresía.
func (f *Feature) InstanceOwnerID(ctx context.Context) (uuid.UUID, error) {
	owner, err := f.InstanceOwner(ctx)
	return owner.ID, err
}

// InstanceOwner devuelve al dueño de la instancia, o un UserRef vacío si todavía
// no existe. Los tokens de agente actúan con su membresía y toman a su nombre.
func (f *Feature) InstanceOwner(ctx context.Context) (kernel.UserRef, error) {
	return instanceowner.New(f.repo).Handle(ctx)
}

// IsUserToken indica si el token es de usuario y no de agente.
func IsUserToken(token string) bool {
	return domain.IsAPIToken(token)
}

// Checker permite a otras features saber si un nombre ya es username y resolver
// usuarios por username o por ID.
type Checker struct {
	repo *store.Users
}

// NewChecker crea el verificador.
func NewChecker(db store.DB) *Checker {
	return &Checker{repo: store.NewUsers(db)}
}

// Exists indica si hay un usuario con ese username.
func (c *Checker) Exists(ctx context.Context, username string) (bool, error) {
	_, err := c.repo.UserByUsername(ctx, strings.TrimSpace(username))
	if errors.Is(err, domain.ErrUserNotFound) {
		return false, nil
	}
	return err == nil, err
}

// Active indica si hay un usuario activo con exactamente ese username: puede ser
// TL o dev de una etapa.
func (c *Checker) Active(ctx context.Context, username string) (bool, error) {
	user, err := c.repo.UserByUsername(ctx, username)
	if errors.Is(err, domain.ErrUserNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !user.Disabled, nil
}

// FindByUsername devuelve el usuario con exactamente ese username, activo o no.
// found es false si no existe.
func (c *Checker) FindByUsername(ctx context.Context, username string) (kernel.UserRef, bool, error) {
	user, err := c.repo.UserByUsername(ctx, username)
	return ref(user, err)
}

// FindByID devuelve el usuario con ese ID, activo o no. found es false si no existe.
func (c *Checker) FindByID(ctx context.Context, id uuid.UUID) (kernel.UserRef, bool, error) {
	user, err := c.repo.UserByID(ctx, id)
	return ref(user, err)
}

func ref(user domain.User, err error) (kernel.UserRef, bool, error) {
	if errors.Is(err, domain.ErrUserNotFound) {
		return kernel.UserRef{}, false, nil
	}
	if err != nil {
		return kernel.UserRef{}, false, err
	}
	return user.Ref(), true, nil
}

// IsPublic indica si la petición va a una ruta de la feature que no pide sesión:
// login, logout, consultar o aceptar una invitación, y pedir o sondear el código
// del CLI pmt.
func IsPublic(r *http.Request) bool {
	path := r.URL.Path
	if r.Method == http.MethodPost && (path == "/api/auth/login" || path == "/api/auth/logout" ||
		path == "/api/cli/login" || path == "/api/cli/login/poll") {
		return true
	}
	rest, ok := strings.CutPrefix(path, "/api/invitations/")
	if !ok || rest == "" {
		return false
	}
	parts := strings.Split(rest, "/")
	return (r.Method == http.MethodGet && len(parts) == 1) ||
		(r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "accept")
}
