package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrInvitationNotFound indica que no hay invitación con ese token.
	ErrInvitationNotFound = kernel.NotFound("invitación no encontrada")
	// ErrInvitationExpired indica que la invitación venció o ya se usó.
	ErrInvitationExpired = kernel.Invalid("la invitación ya no es válida")
)

// Invitation es una invitación del dueño a un correo.
type Invitation struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at"`
}

// InvitationStatus es lo que ve quien abre el enlace, sin sesión.
type InvitationStatus struct {
	Email string `json:"email"`
	Valid bool   `json:"valid"`
}

// InvitationURL arma el enlace que el dueño manda al invitado.
func InvitationURL(base, token string) string {
	return strings.TrimRight(base, "/") + "/invitacion/" + token
}

// InvitationValid indica si una invitación todavía se puede aceptar.
func InvitationValid(expires time.Time, accepted, userExists bool, now time.Time) bool {
	return !accepted && expires.After(now) && !userExists
}
