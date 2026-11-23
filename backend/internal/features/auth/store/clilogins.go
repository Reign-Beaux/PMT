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
)

// StartCLILogin borra los códigos vencidos y guarda uno nuevo si los pendientes
// no llegan a maxPending. Un código repetido responde domain.ErrCLICodeTaken.
func (s *Users) StartCLILogin(ctx context.Context, login domain.NewCLILogin, maxPending int) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.DeleteExpiredCLILogins(ctx); err != nil {
			return fmt.Errorf("borrar códigos vencidos: %w", err)
		}
		pending, err := q.CountPendingCLILogins(ctx)
		if err != nil {
			return fmt.Errorf("contar códigos pendientes: %w", err)
		}
		if pending >= int64(maxPending) {
			return domain.ErrTooManyCLILogins
		}
		err = q.CreateCLILogin(ctx, sqlcgen.CreateCLILoginParams{
			ID: login.ID, Code: login.Code, PollHash: login.PollHash, Machine: login.Machine, ExpiresAt: login.ExpiresAt,
		})
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "cli_logins_code_key" {
			return domain.ErrCLICodeTaken
		}
		if err != nil {
			return fmt.Errorf("crear código: %w", err)
		}
		return nil
	})
}

// CLILoginByCode devuelve el inicio de sesión de un código.
func (s *Users) CLILoginByCode(ctx context.Context, code string) (domain.CLILogin, error) {
	row, err := s.queries().GetCLILoginByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CLILogin{}, domain.ErrCLILoginNotFound
	}
	if err != nil {
		return domain.CLILogin{}, fmt.Errorf("obtener código: %w", err)
	}
	return domain.CLILogin{ID: row.ID, Code: row.Code, Machine: row.Machine, State: row.Status, ExpiresAt: row.ExpiresAt}, nil
}

// AuthorizeCLILogin marca autorizado un código pendiente y vigente. Devuelve si lo marcó.
func (s *Users) AuthorizeCLILogin(ctx context.Context, code string, userID uuid.UUID) (bool, error) {
	n, err := s.queries().AuthorizeCLILogin(ctx, sqlcgen.AuthorizeCLILoginParams{Code: code, UserID: pgtype.UUID{Bytes: userID, Valid: true}})
	if err != nil {
		return false, fmt.Errorf("autorizar código: %w", err)
	}
	return n > 0, nil
}

// CancelCLILogin marca cancelado un código pendiente y vigente. Devuelve si lo marcó.
func (s *Users) CancelCLILogin(ctx context.Context, code string) (bool, error) {
	n, err := s.queries().CancelCLILogin(ctx, code)
	if err != nil {
		return false, fmt.Errorf("cancelar código: %w", err)
	}
	return n > 0, nil
}

// PollCLILogin dice en qué va el código del secreto. Si está autorizado y
// vigente, en una transacción crea el token del usuario con el nombre de la
// máquina y deja el código entregado: el token se entrega una sola vez. Si el
// usuario se desactivó entre autorizar y sondear, responde cancelado.
func (s *Users) PollCLILogin(ctx context.Context, pollHash []byte, token domain.IssuedToken, now time.Time) (domain.CLIPoll, error) {
	var poll domain.CLIPoll
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.LockCLILoginByPollHash(ctx, pollHash)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrCLILoginNotFound
		}
		if err != nil {
			return fmt.Errorf("obtener código: %w", err)
		}
		poll.Status = domain.CLILoginStatus(row.Status, row.ExpiresAt, now)
		if poll.Status != domain.CLIStatusAuthorized {
			return nil
		}
		user, err := q.GetUserByID(ctx, row.UserID.Bytes)
		if err != nil {
			return notFound(err, "obtener usuario")
		}
		if user.DisabledAt.Valid {
			poll.Status = domain.CLIStatusCancelled
			return nil
		}
		if _, err = q.CreateAPIToken(ctx, sqlcgen.CreateAPITokenParams{
			ID: token.ID, UserID: user.ID, Name: domain.CLITokenName(row.Machine), TokenHash: domain.HashToken(token.Token),
		}); err != nil {
			return fmt.Errorf("crear token: %w", err)
		}
		if err = q.ConsumeCLILogin(ctx, sqlcgen.ConsumeCLILoginParams{ID: row.ID, TokenID: pgtype.UUID{Bytes: token.ID, Valid: true}}); err != nil {
			return fmt.Errorf("entregar código: %w", err)
		}
		poll.UserID, poll.Username, poll.TokenID = user.ID, user.Username, token.ID
		return nil
	})
	if err != nil {
		return domain.CLIPoll{}, err
	}
	return poll, nil
}
