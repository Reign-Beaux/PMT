// Package authtest tiene un repositorio en memoria para las pruebas de las slices
// de auth. Reproduce las reglas de la base: unicidad, usuarios activos, sesiones
// vigentes y tokens sin revocar.
package authtest

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

type user struct {
	domain.User
	hash string
}

type session struct {
	hash    []byte
	userID  uuid.UUID
	expires time.Time
}

type token struct {
	domain.APIToken
	userID  uuid.UUID
	hash    []byte
	revoked bool
}

type invitation struct {
	domain.Invitation
	hash []byte
}

// Repo es el repositorio en memoria.
type Repo struct {
	mu          sync.Mutex
	users       []*user
	sessions    []session
	tokens      []*token
	invitations []*invitation
	cliLogins   []*cliLogin
}

// New crea un repositorio vacío.
func New() *Repo { return &Repo{} }

// AddUser agrega un usuario con su contraseña y lo devuelve.
func (r *Repo) AddUser(email, username string, owner bool, password string) domain.User {
	hash, err := domain.HashPassword(password)
	if err != nil {
		panic(err)
	}
	u := &user{User: domain.User{ID: uuid.New(), Email: email, Username: username, Name: username, IsInstanceOwner: owner, CreatedAt: time.Now()}, hash: hash}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users = append(r.users, u)
	return u.User
}

// Sessions cuenta las sesiones guardadas.
func (r *Repo) Sessions() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.sessions) }

// Invitations devuelve las invitaciones guardadas.
func (r *Repo) Invitations() []domain.Invitation {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Invitation{}
	for _, i := range r.invitations {
		out = append(out, i.Invitation)
	}
	return out
}

// AddInvitation guarda una invitación con el token dado.
func (r *Repo) AddInvitation(email, tok string, expires time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invitations = append(r.invitations, &invitation{Invitation: domain.Invitation{ID: uuid.New(), Email: email, ExpiresAt: expires}, hash: domain.HashToken(tok)})
}

func (r *Repo) find(match func(*user) bool) *user {
	for _, u := range r.users {
		if match(u) {
			return u
		}
	}
	return nil
}

func (r *Repo) HasUsers(context.Context) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.users) > 0, nil
}

func (r *Repo) CreateOwner(_ context.Context, u domain.User, hash string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.users) > 0 {
		return false, nil
	}
	r.users = append(r.users, &user{User: u, hash: hash})
	return true, nil
}

func (r *Repo) UserByEmail(_ context.Context, email string) (domain.User, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u := r.find(func(u *user) bool { return u.Email == email }); u != nil {
		return u.User, u.hash, nil
	}
	return domain.User{}, "", domain.ErrUserNotFound
}

func (r *Repo) UserByUsername(_ context.Context, username string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u := r.find(func(u *user) bool { return u.Username == username }); u != nil {
		return u.User, nil
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (r *Repo) UserByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u := r.find(func(u *user) bool { return u.ID == id }); u != nil {
		return u.User, nil
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (r *Repo) InstanceOwner(_ context.Context) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u := r.find(func(u *user) bool { return u.IsInstanceOwner }); u != nil {
		return u.User, nil
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (r *Repo) PasswordHash(_ context.Context, id uuid.UUID) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u := r.find(func(u *user) bool { return u.ID == id }); u != nil {
		return u.hash, nil
	}
	return "", domain.ErrUserNotFound
}

func (r *Repo) ListUsers(context.Context) ([]domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.User{}
	for _, u := range r.users {
		out = append(out, u.User)
	}
	return out, nil
}

func (r *Repo) CreateSession(_ context.Context, hash []byte, userID uuid.UUID, expires time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions = append(r.sessions, session{hash, userID, expires})
	return nil
}

func (r *Repo) UserBySession(_ context.Context, hash []byte) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sessions {
		if bytes.Equal(s.hash, hash) && s.expires.After(time.Now()) {
			if u := r.find(func(u *user) bool { return u.ID == s.userID && !u.Disabled }); u != nil {
				return u.User, nil
			}
		}
	}
	return domain.User{}, kernel.ErrSessionInvalid
}

func (r *Repo) DeleteSession(_ context.Context, hash []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.sessions[:0]
	for _, s := range r.sessions {
		if !bytes.Equal(s.hash, hash) {
			kept = append(kept, s)
		}
	}
	r.sessions = kept
	return nil
}

func (r *Repo) ChangePassword(_ context.Context, id uuid.UUID, hash string, keep []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := r.find(func(u *user) bool { return u.ID == id })
	if u == nil {
		return domain.ErrUserNotFound
	}
	u.hash = hash
	kept := r.sessions[:0]
	for _, s := range r.sessions {
		if s.userID != id || bytes.Equal(s.hash, keep) {
			kept = append(kept, s)
		}
	}
	r.sessions = kept
	return nil
}

func (r *Repo) DisableUser(_ context.Context, id uuid.UUID) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := r.find(func(u *user) bool { return u.ID == id })
	if u == nil {
		return domain.User{}, domain.ErrUserNotFound
	}
	u.Disabled = true
	kept := r.sessions[:0]
	for _, s := range r.sessions {
		if s.userID != id {
			kept = append(kept, s)
		}
	}
	r.sessions = kept
	for _, t := range r.tokens {
		if t.userID == id {
			t.revoked = true
		}
	}
	return u.User, nil
}

func (r *Repo) EnableUser(_ context.Context, id uuid.UUID) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := r.find(func(u *user) bool { return u.ID == id })
	if u == nil {
		return domain.User{}, domain.ErrUserNotFound
	}
	u.Disabled = false
	return u.User, nil
}

func (r *Repo) CreateAPIToken(_ context.Context, id, userID uuid.UUID, name string, hash []byte) (time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	r.tokens = append(r.tokens, &token{APIToken: domain.APIToken{ID: id, Name: name, CreatedAt: now}, userID: userID, hash: hash})
	return now, nil
}

func (r *Repo) ListAPITokens(_ context.Context, userID uuid.UUID) ([]domain.APIToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.APIToken{}
	for _, t := range r.tokens {
		if t.userID == userID && !t.revoked {
			out = append(out, t.APIToken)
		}
	}
	return out, nil
}

func (r *Repo) RevokeAPIToken(_ context.Context, id, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tokens {
		if t.ID == id && t.userID == userID && !t.revoked {
			t.revoked = true
			return nil
		}
	}
	return domain.ErrTokenNotFound
}

func (r *Repo) UserByAPIToken(_ context.Context, hash []byte) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tokens {
		if bytes.Equal(t.hash, hash) && !t.revoked {
			if u := r.find(func(u *user) bool { return u.ID == t.userID && !u.Disabled }); u != nil {
				now := time.Now()
				t.LastUsedAt = &now
				return u.User, nil
			}
		}
	}
	return domain.User{}, kernel.ErrUnauthenticated
}

func (r *Repo) CreateInvitation(_ context.Context, id uuid.UUID, email string, hash []byte, _ uuid.UUID, expires time.Time) (time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invitations = append(r.invitations, &invitation{Invitation: domain.Invitation{ID: id, Email: email, ExpiresAt: expires}, hash: hash})
	return expires, nil
}

func (r *Repo) ListInvitations(context.Context) ([]domain.Invitation, error) {
	return r.Invitations(), nil
}

func (r *Repo) InvitationStatus(_ context.Context, hash []byte, now time.Time) (domain.InvitationStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, i := range r.invitations {
		if bytes.Equal(i.hash, hash) {
			exists := r.find(func(u *user) bool { return u.Email == i.Email }) != nil
			return domain.InvitationStatus{Email: i.Email, Valid: domain.InvitationValid(i.ExpiresAt, i.AcceptedAt != nil, exists, now)}, nil
		}
	}
	return domain.InvitationStatus{}, domain.ErrInvitationNotFound
}

func (r *Repo) AcceptInvitation(_ context.Context, hash []byte, u domain.User, passwordHash string, sessionHash []byte, expires time.Time) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, i := range r.invitations {
		if !bytes.Equal(i.hash, hash) {
			continue
		}
		if i.AcceptedAt != nil || !i.ExpiresAt.After(time.Now()) {
			return domain.User{}, domain.ErrInvitationExpired
		}
		if r.find(func(x *user) bool { return x.Email == i.Email }) != nil {
			return domain.User{}, domain.ErrInvitationExpired
		}
		if r.find(func(x *user) bool { return x.Username == u.Username }) != nil {
			return domain.User{}, domain.ErrUsernameTaken
		}
		u.Email = i.Email
		u.CreatedAt = time.Now()
		r.users = append(r.users, &user{User: u, hash: passwordHash})
		now := time.Now()
		i.AcceptedAt = &now
		r.sessions = append(r.sessions, session{sessionHash, u.ID, expires})
		return u, nil
	}
	return domain.User{}, domain.ErrInvitationNotFound
}

// Agents simula los nombres de agentes existentes.
type Agents map[string]bool

// Exists indica si el nombre es de un agente.
func (a Agents) Exists(_ context.Context, name string) (bool, error) { return a[name], nil }

type cliLogin struct {
	domain.CLILogin
	pollHash []byte
	userID   uuid.UUID
}

// CLILogins devuelve los inicios de sesión del CLI guardados.
func (r *Repo) CLILogins() []domain.CLILogin {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.CLILogin{}
	for _, l := range r.cliLogins {
		out = append(out, l.CLILogin)
	}
	return out
}

// ExpireCLILogins vence todos los códigos, como si pasaran diez minutos.
func (r *Repo) ExpireCLILogins() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.cliLogins {
		l.ExpiresAt = time.Now().Add(-time.Second)
	}
}

func (r *Repo) StartCLILogin(_ context.Context, login domain.NewCLILogin, maxPending int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	kept := r.cliLogins[:0]
	pending := 0
	for _, l := range r.cliLogins {
		if !l.ExpiresAt.After(now) {
			continue
		}
		if l.Code == login.Code {
			return domain.ErrCLICodeTaken
		}
		if l.State == domain.CLIStatePending {
			pending++
		}
		kept = append(kept, l)
	}
	r.cliLogins = kept
	if pending >= maxPending {
		return domain.ErrTooManyCLILogins
	}
	r.cliLogins = append(r.cliLogins, &cliLogin{
		CLILogin: domain.CLILogin{ID: login.ID, Code: login.Code, Machine: login.Machine, State: domain.CLIStatePending, ExpiresAt: login.ExpiresAt},
		pollHash: login.PollHash,
	})
	return nil
}

func (r *Repo) CLILoginByCode(_ context.Context, code string) (domain.CLILogin, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.cliLogins {
		if l.Code == code {
			return l.CLILogin, nil
		}
	}
	return domain.CLILogin{}, domain.ErrCLILoginNotFound
}

func (r *Repo) decideCLILogin(code, state string, userID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.cliLogins {
		if l.Code == code && l.State == domain.CLIStatePending && l.ExpiresAt.After(time.Now()) {
			l.State, l.userID = state, userID
			return true
		}
	}
	return false
}

func (r *Repo) AuthorizeCLILogin(_ context.Context, code string, userID uuid.UUID) (bool, error) {
	return r.decideCLILogin(code, domain.CLIStateAuthorized, userID), nil
}

func (r *Repo) CancelCLILogin(_ context.Context, code string) (bool, error) {
	return r.decideCLILogin(code, domain.CLIStateCancelled, uuid.Nil), nil
}

func (r *Repo) PollCLILogin(_ context.Context, pollHash []byte, tok domain.IssuedToken, now time.Time) (domain.CLIPoll, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.cliLogins {
		if !bytes.Equal(l.pollHash, pollHash) {
			continue
		}
		poll := domain.CLIPoll{Status: domain.CLILoginStatus(l.State, l.ExpiresAt, now)}
		if poll.Status != domain.CLIStatusAuthorized {
			return poll, nil
		}
		u := r.find(func(u *user) bool { return u.ID == l.userID })
		if u == nil {
			return domain.CLIPoll{}, domain.ErrUserNotFound
		}
		if u.Disabled {
			return domain.CLIPoll{Status: domain.CLIStatusCancelled}, nil
		}
		r.tokens = append(r.tokens, &token{APIToken: domain.APIToken{ID: tok.ID, Name: domain.CLITokenName(l.Machine), CreatedAt: now}, userID: u.ID, hash: domain.HashToken(tok.Token)})
		l.State = domain.CLIStateConsumed
		poll.UserID, poll.Username, poll.TokenID = u.ID, u.Username, tok.ID
		return poll, nil
	}
	return domain.CLIPoll{}, domain.ErrCLILoginNotFound
}
