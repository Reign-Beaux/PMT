// Package store implementa la persistencia de la feature agents.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"pmt/internal/features/agents/domain"
	"pmt/internal/features/agents/store/sqlcgen"
	"pmt/internal/kernel"
	"pmt/internal/platform/postgres"
)

// Agents implementa con PostgreSQL las interfaces que piden las slices.
type Agents struct {
	queries *sqlcgen.Queries
}

// NewAgents crea el repositorio de agentes.
func NewAgents(db sqlcgen.DBTX) *Agents {
	return &Agents{queries: sqlcgen.New(db)}
}

// Create guarda un agente nuevo con el hash de su token.
func (s *Agents) Create(ctx context.Context, agent domain.Agent, tokenHash []byte) (domain.Agent, error) {
	row, err := s.queries.CreateAgent(ctx, sqlcgen.CreateAgentParams{
		ID:        agent.ID,
		Name:      agent.Name,
		Profile:   string(agent.Profile),
		TokenHash: tokenHash,
	})
	if postgres.IsUniqueViolation(err) {
		return domain.Agent{}, domain.ErrAgentNameTaken
	}
	if err != nil {
		return domain.Agent{}, fmt.Errorf("crear agente: %w", err)
	}
	return toDomain(row), nil
}

// GetByName busca un agente por su nombre, revocado o no.
func (s *Agents) GetByName(ctx context.Context, name string) (domain.Agent, error) {
	row, err := s.queries.GetAgentByName(ctx, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, domain.ErrAgentNotFound
	}
	if err != nil {
		return domain.Agent{}, fmt.Errorf("obtener agente: %w", err)
	}
	return toDomain(row), nil
}

// GetActiveByTokenHash busca el agente activo de un token.
func (s *Agents) GetActiveByTokenHash(ctx context.Context, tokenHash []byte) (domain.Agent, error) {
	row, err := s.queries.GetActiveAgentByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, kernel.ErrUnauthenticated
	}
	if err != nil {
		return domain.Agent{}, fmt.Errorf("autenticar agente: %w", err)
	}
	return toDomain(row), nil
}

// List devuelve los agentes por nombre.
func (s *Agents) List(ctx context.Context) ([]domain.Agent, error) {
	rows, err := s.queries.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar agentes: %w", err)
	}
	agents := make([]domain.Agent, 0, len(rows))
	for _, row := range rows {
		agents = append(agents, toDomain(row))
	}
	return agents, nil
}

// UpdateTokenHash sustituye el token de un agente activo.
func (s *Agents) UpdateTokenHash(ctx context.Context, id uuid.UUID, tokenHash []byte) error {
	updated, err := s.queries.UpdateAgentTokenHash(ctx, sqlcgen.UpdateAgentTokenHashParams{ID: id, TokenHash: tokenHash})
	if err != nil {
		return fmt.Errorf("rotar token: %w", err)
	}
	if updated == 0 {
		return domain.ErrAgentRevoked
	}
	return nil
}

// Revoke revoca un agente activo.
func (s *Agents) Revoke(ctx context.Context, id uuid.UUID) error {
	revoked, err := s.queries.RevokeAgent(ctx, id)
	if err != nil {
		return fmt.Errorf("revocar agente: %w", err)
	}
	if revoked == 0 {
		return domain.ErrAgentRevoked
	}
	return nil
}

func revokedAt(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func toDomain(row sqlcgen.Agent) domain.Agent {
	return domain.Agent{
		ID:        row.ID,
		Name:      row.Name,
		Profile:   kernel.Profile(row.Profile),
		RevokedAt: revokedAt(row.RevokedAt),
		CreatedAt: row.CreatedAt,
	}
}
