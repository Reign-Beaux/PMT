package revoke

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/agents/domain"
)

type fakeRepo struct {
	agent   domain.Agent
	err     error
	revoked uuid.UUID
}

func (r *fakeRepo) GetByName(context.Context, string) (domain.Agent, error) {
	return r.agent, r.err
}

func (r *fakeRepo) Revoke(_ context.Context, id uuid.UUID) error {
	r.revoked = id
	return nil
}

func TestHandleRevokes(t *testing.T) {
	repo := &fakeRepo{agent: domain.Agent{ID: uuid.New()}}

	if err := New(repo).Handle(context.Background(), "dev"); err != nil {
		t.Fatal(err)
	}
	if repo.revoked != repo.agent.ID {
		t.Errorf("revocado = %v", repo.revoked)
	}
}

func TestHandleMissing(t *testing.T) {
	repo := &fakeRepo{err: domain.ErrAgentNotFound}

	if err := New(repo).Handle(context.Background(), "x"); !errors.Is(err, domain.ErrAgentNotFound) {
		t.Errorf("error = %v", err)
	}
}
