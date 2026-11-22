// Package domain contiene los usuarios, sus sesiones, sus tokens de API y las
// invitaciones.
package domain

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrUserNotFound indica que el usuario no existe.
	ErrUserNotFound = kernel.NotFound("usuario no encontrado")
	// ErrUsersOnly indica que la operación es de usuarios y actúa un agente.
	ErrUsersOnly = kernel.Forbidden("solo los usuarios pueden usar esta operación")
	// ErrProfileDeclared indica que actúa un usuario con un perfil declarado: las
	// operaciones de cuenta son del perfil User.
	ErrProfileDeclared = kernel.Forbidden("las operaciones de cuenta se usan sin perfil declarado")
	// ErrOwnerOnly indica que la operación es del dueño de la instancia.
	ErrOwnerOnly = kernel.Forbidden("solo el dueño de la instancia puede usar esta operación")
	// ErrUsernameInvalid indica que el username no cumple el formato.
	ErrUsernameInvalid = kernel.Invalid("username inválido: de 1 a 40 caracteres en minúsculas, números y guiones, sin guion al inicio ni al final")
	// ErrUsernameTaken indica que el username ya es de un usuario o de un agente.
	ErrUsernameTaken = kernel.Conflict("el username ya está en uso")
	// ErrNameRequired indica que el nombre está vacío.
	ErrNameRequired = kernel.Invalid("el nombre es obligatorio")
	// ErrEmailInvalid indica que el correo no tiene forma de correo.
	ErrEmailInvalid = kernel.Invalid("correo inválido")
	// ErrEmailTaken indica que el correo ya está registrado.
	ErrEmailTaken = kernel.Conflict("el correo ya está registrado")
	// ErrPasswordTooShort indica que la contraseña tiene menos de 8 caracteres.
	ErrPasswordTooShort = kernel.Invalid("la contraseña debe tener al menos 8 caracteres")
	// ErrOwnerCannotBeDisabled indica que se intentó desactivar al dueño.
	ErrOwnerCannotBeDisabled = kernel.Conflict("el dueño de la instancia no se puede desactivar")
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$`)

// User es una persona que entra a PMT. Su username es el nombre con el que actúa
// y no cambia.
type User struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	Name            string    `json:"name"`
	IsInstanceOwner bool      `json:"is_instance_owner"`
	Disabled        bool      `json:"disabled"`
	CreatedAt       time.Time `json:"created_at"`
}

// Actor devuelve la identidad con la que actúa el usuario sin perfil declarado:
// User, con su ID para la membresía de los proyectos. El perfil que declare la
// sesión de un token lo aplica kernel.AsUser.
func (u User) Actor() kernel.Actor {
	return kernel.Actor{Name: u.Username, Profile: kernel.ProfileUser, UserID: u.ID, UserName: u.Username}
}

// Ref devuelve el usuario como lo ven otras features.
func (u User) Ref() kernel.UserRef {
	return kernel.UserRef{ID: u.ID, Username: u.Username, Name: u.Name, Active: !u.Disabled}
}

// NormalizeEmail quita espacios y pasa el correo a minúsculas.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail normaliza el correo y comprueba que tenga forma de correo.
func ValidateEmail(email string) (string, error) {
	email = NormalizeEmail(email)
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || domain == "" || strings.ContainsAny(email, " \t\r\n") {
		return "", ErrEmailInvalid
	}
	return email, nil
}

// ValidateUsername comprueba el formato del username. No lo normaliza: el
// usuario lo escribe tal como quedará.
func ValidateUsername(username string) (string, error) {
	if !usernamePattern.MatchString(username) {
		return "", ErrUsernameInvalid
	}
	return username, nil
}

// ValidateName quita espacios y exige un nombre.
func ValidateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrNameRequired
	}
	return name, nil
}

// ValidatePassword exige al menos 8 caracteres.
func ValidatePassword(password string) error {
	if len([]rune(password)) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}

// UserFinder busca un usuario por su username. Devuelve ErrUserNotFound si no existe.
type UserFinder interface {
	UserByUsername(ctx context.Context, username string) (User, error)
}

// AgentNames indica si un nombre ya es de un agente. Lo cumple la feature agents.
type AgentNames interface {
	Exists(ctx context.Context, name string) (bool, error)
}

// CurrentUser devuelve el usuario que actúa con el perfil User, para las
// operaciones de cuenta. Un agente no es usuario; un usuario con perfil declarado
// recibe ErrProfileDeclared.
func CurrentUser(ctx context.Context, finder UserFinder, actor kernel.Actor) (User, error) {
	if actor.Name == "" {
		return User{}, ErrUsersOnly
	}
	user, err := finder.UserByUsername(ctx, actor.Name)
	if errors.Is(err, ErrUserNotFound) {
		return User{}, ErrUsersOnly
	}
	if err != nil {
		return User{}, err
	}
	if actor.Profile != kernel.ProfileUser {
		return User{}, ErrProfileDeclared
	}
	return user, nil
}

// CurrentOwner devuelve el usuario que actúa si es el dueño de la instancia.
func CurrentOwner(ctx context.Context, finder UserFinder, actor kernel.Actor) (User, error) {
	user, err := CurrentUser(ctx, finder, actor)
	if err != nil {
		return User{}, err
	}
	if !user.IsInstanceOwner {
		return User{}, ErrOwnerOnly
	}
	return user, nil
}

// EnsureUsernameFree rechaza un username que ya sea el nombre de un agente. La
// unicidad entre usuarios la garantiza la base.
func EnsureUsernameFree(ctx context.Context, agents AgentNames, username string) error {
	taken, err := agents.Exists(ctx, username)
	if err != nil {
		return err
	}
	if taken {
		return ErrUsernameTaken
	}
	return nil
}
