// Package store implementa la persistencia de la feature auth.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"pmt/internal/features/auth/domain"
	"pmt/internal/features/auth/store/sqlcgen"
	"pmt/internal/kernel"
)

// DB es la base de datos de la feature: consultas y transacciones. pgxpool.Pool la implementa.
type DB interface {
	sqlcgen.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Users implementa con PostgreSQL las interfaces que piden las slices.
type Users struct {
	db DB
}

// NewUsers crea el repositorio de usuarios.
func NewUsers(db DB) *Users {
	return &Users{db: db}
}

func (s *Users) queries() *sqlcgen.Queries {
	return sqlcgen.New(s.db)
}

// HasUsers indica si ya existe algún usuario.
func (s *Users) HasUsers(ctx context.Context) (bool, error) {
	exists, err := s.queries().HasUsers(ctx)
	if err != nil {
		return false, fmt.Errorf("contar usuarios: %w", err)
	}
	return exists, nil
}

// CreateOwner crea al dueño de la instancia solo si no hay usuarios. Devuelve si lo creó.
func (s *Users) CreateOwner(ctx context.Context, user domain.User, passwordHash string) (bool, error) {
	n, err := s.queries().CreateOwnerIfNoUsers(ctx, sqlcgen.CreateOwnerIfNoUsersParams{
		ID: user.ID, Email: user.Email, Username: user.Username, Name: user.Name, PasswordHash: passwordHash,
	})
	if err != nil {
		return false, fmt.Errorf("crear dueño: %w", err)
	}
	return n > 0, nil
}

// UserByEmail devuelve el usuario y el hash de su contraseña.
func (s *Users) UserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	row, err := s.queries().GetUserByEmail(ctx, email)
	if err != nil {
		return domain.User{}, "", notFound(err, "obtener usuario")
	}
	return toDomain(row), row.PasswordHash, nil
}

// UserByUsername devuelve el usuario de un username.
func (s *Users) UserByUsername(ctx context.Context, username string) (domain.User, error) {
	row, err := s.queries().GetUserByUsername(ctx, username)
	if err != nil {
		return domain.User{}, notFound(err, "obtener usuario")
	}
	return toDomain(row), nil
}

// UserByID devuelve el usuario de un ID.
func (s *Users) UserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := s.queries().GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, notFound(err, "obtener usuario")
	}
	return toDomain(row), nil
}

// PasswordHash devuelve el hash de la contraseña de un usuario.
func (s *Users) PasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	row, err := s.queries().GetUserByID(ctx, id)
	if err != nil {
		return "", notFound(err, "obtener contraseña")
	}
	return row.PasswordHash, nil
}

// ListUsers devuelve los usuarios por username.
func (s *Users) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.queries().ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar usuarios: %w", err)
	}
	users := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, toDomain(row))
	}
	return users, nil
}

// CreateSession guarda el hash de una sesión nueva.
func (s *Users) CreateSession(ctx context.Context, sessionHash []byte, userID uuid.UUID, expires time.Time) error {
	err := s.queries().CreateSession(ctx, sqlcgen.CreateSessionParams{ID: sessionHash, UserID: userID, ExpiresAt: expires})
	if err != nil {
		return fmt.Errorf("crear sesión: %w", err)
	}
	return nil
}

// UserBySession devuelve el usuario activo de una sesión vigente.
func (s *Users) UserBySession(ctx context.Context, sessionHash []byte) (domain.User, error) {
	row, err := s.queries().GetUserBySession(ctx, sessionHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, kernel.ErrSessionInvalid
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("autenticar sesión: %w", err)
	}
	return toDomain(row), nil
}

// DeleteSession borra una sesión.
func (s *Users) DeleteSession(ctx context.Context, sessionHash []byte) error {
	if err := s.queries().DeleteSession(ctx, sessionHash); err != nil {
		return fmt.Errorf("borrar sesión: %w", err)
	}
	return nil
}

// ChangePassword guarda el hash nuevo y cierra las demás sesiones del usuario.
func (s *Users) ChangePassword(ctx context.Context, id uuid.UUID, passwordHash string, keepSession []byte) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.SetPasswordHash(ctx, sqlcgen.SetPasswordHashParams{ID: id, PasswordHash: passwordHash}); err != nil {
			return fmt.Errorf("cambiar contraseña: %w", err)
		}
		if err := q.DeleteOtherSessions(ctx, sqlcgen.DeleteOtherSessionsParams{UserID: id, ID: keepSession}); err != nil {
			return fmt.Errorf("cerrar sesiones: %w", err)
		}
		return nil
	})
}

// DisableUser desactiva al usuario, borra sus sesiones y revoca sus tokens.
func (s *Users) DisableUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	var user domain.User
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.DisableUser(ctx, id)
		if err != nil {
			return notFound(err, "desactivar usuario")
		}
		if err = q.DeleteUserSessions(ctx, id); err != nil {
			return fmt.Errorf("cerrar sesiones: %w", err)
		}
		if err = q.RevokeUserAPITokens(ctx, id); err != nil {
			return fmt.Errorf("revocar tokens: %w", err)
		}
		user = toDomain(row)
		return nil
	})
	return user, err
}

// EnableUser reactiva al usuario. Sus tokens revocados siguen revocados.
func (s *Users) EnableUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := s.queries().EnableUser(ctx, id)
	if err != nil {
		return domain.User{}, notFound(err, "reactivar usuario")
	}
	return toDomain(row), nil
}

// CreateAPIToken guarda el hash de un token nuevo y devuelve su fecha de creación.
func (s *Users) CreateAPIToken(ctx context.Context, id, userID uuid.UUID, name string, tokenHash []byte) (time.Time, error) {
	created, err := s.queries().CreateAPIToken(ctx, sqlcgen.CreateAPITokenParams{ID: id, UserID: userID, Name: name, TokenHash: tokenHash})
	if err != nil {
		return time.Time{}, fmt.Errorf("crear token: %w", err)
	}
	return created, nil
}

// ListAPITokens devuelve los tokens vigentes del usuario, del más nuevo al más viejo.
func (s *Users) ListAPITokens(ctx context.Context, userID uuid.UUID) ([]domain.APIToken, error) {
	rows, err := s.queries().ListAPITokens(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listar tokens: %w", err)
	}
	tokens := make([]domain.APIToken, 0, len(rows))
	for _, row := range rows {
		tokens = append(tokens, domain.APIToken{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt, LastUsedAt: timePtr(row.LastUsedAt)})
	}
	return tokens, nil
}

// RevokeAPIToken revoca un token vigente del usuario.
func (s *Users) RevokeAPIToken(ctx context.Context, id, userID uuid.UUID) error {
	n, err := s.queries().RevokeAPIToken(ctx, sqlcgen.RevokeAPITokenParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("revocar token: %w", err)
	}
	if n == 0 {
		return domain.ErrTokenNotFound
	}
	return nil
}

// UserByAPIToken devuelve el usuario activo de un token vigente y anota su uso.
func (s *Users) UserByAPIToken(ctx context.Context, tokenHash []byte) (domain.User, error) {
	row, err := s.queries().TouchAPIToken(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, kernel.ErrUnauthenticated
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("autenticar token: %w", err)
	}
	return toDomain(row), nil
}

// CreateInvitation guarda el hash de una invitación y devuelve su vencimiento.
func (s *Users) CreateInvitation(ctx context.Context, id uuid.UUID, email string, tokenHash []byte, invitedBy uuid.UUID, expires time.Time) (time.Time, error) {
	at, err := s.queries().CreateInvitation(ctx, sqlcgen.CreateInvitationParams{
		ID: id, Email: email, TokenHash: tokenHash, InvitedBy: invitedBy, ExpiresAt: expires,
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("crear invitación: %w", err)
	}
	return at, nil
}

// ListInvitations devuelve las invitaciones, de la más nueva a la más vieja.
func (s *Users) ListInvitations(ctx context.Context) ([]domain.Invitation, error) {
	rows, err := s.queries().ListInvitations(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar invitaciones: %w", err)
	}
	items := make([]domain.Invitation, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.Invitation{ID: row.ID, Email: row.Email, ExpiresAt: row.ExpiresAt, AcceptedAt: timePtr(row.AcceptedAt)})
	}
	return items, nil
}

// InvitationStatus dice a qué correo es la invitación y si todavía vale.
func (s *Users) InvitationStatus(ctx context.Context, tokenHash []byte, now time.Time) (domain.InvitationStatus, error) {
	row, err := s.queries().GetInvitationByHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.InvitationStatus{}, domain.ErrInvitationNotFound
	}
	if err != nil {
		return domain.InvitationStatus{}, fmt.Errorf("obtener invitación: %w", err)
	}
	return domain.InvitationStatus{
		Email: row.Email,
		Valid: domain.InvitationValid(row.ExpiresAt, row.AcceptedAt.Valid, row.UserExists, now),
	}, nil
}

// AcceptInvitation crea al usuario invitado, marca la invitación y abre su
// sesión en una transacción. El correo es el de la invitación.
func (s *Users) AcceptInvitation(ctx context.Context, tokenHash []byte, user domain.User, passwordHash string, sessionHash []byte, expires time.Time) (domain.User, error) {
	var created domain.User
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		inv, err := q.LockInvitationByHash(ctx, tokenHash)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInvitationNotFound
		}
		if err != nil {
			return fmt.Errorf("obtener invitación: %w", err)
		}
		if inv.AcceptedAt.Valid || !inv.ExpiresAt.After(time.Now()) {
			return domain.ErrInvitationExpired
		}
		row, err := q.CreateInvitedUser(ctx, sqlcgen.CreateInvitedUserParams{
			ID: user.ID, Email: inv.Email, Username: user.Username, Name: user.Name, PasswordHash: passwordHash,
		})
		if err != nil {
			return uniqueUser(err)
		}
		if err = q.MarkInvitationAccepted(ctx, inv.ID); err != nil {
			return fmt.Errorf("marcar invitación: %w", err)
		}
		if err = q.CreateSession(ctx, sqlcgen.CreateSessionParams{ID: sessionHash, UserID: row.ID, ExpiresAt: expires}); err != nil {
			return fmt.Errorf("crear sesión: %w", err)
		}
		created = toDomain(row)
		return nil
	})
	return created, err
}

// uniqueUser traduce la violación de unicidad de users al error del campo repetido.
func uniqueUser(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "users_email_key" {
			return domain.ErrInvitationExpired
		}
		return domain.ErrUsernameTaken
	}
	return fmt.Errorf("crear usuario: %w", err)
}

func notFound(err error, action string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	return fmt.Errorf("%s: %w", action, err)
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func toDomain(row sqlcgen.User) domain.User {
	return domain.User{
		ID:              row.ID,
		Email:           row.Email,
		Username:        row.Username,
		Name:            row.Name,
		IsInstanceOwner: row.IsInstanceOwner,
		Disabled:        row.DisabledAt.Valid,
		CreatedAt:       row.CreatedAt,
	}
}
