package rotate

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/agents/domain"
)

type fakeRepo struct {
	agent domain.Agent
	err   error
	hash  []byte
}

func (r *fakeRepo) GetByName(context.Context, string) (domain.Agent, error) {
	return r.agent, r.err
}

func (r *fakeRepo) UpdateTokenHash(_ context.Context, _ uuid.UUID, tokenHash []byte) error {
	r.hash = tokenHash
	return nil
}

func TestHandleRotates(t *testing.T) {
	repo := &fakeRepo{}

	token, err := New(repo).Handle(context.Background(), "dev")

	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(repo.hash, domain.HashToken(token)) {
		t.Error("debe guardar el hash del token devuelto")
	}
}

func TestHandleRejectsRevoked(t *testing.T) {
	now := time.Now()
	repo := &fakeRepo{agent: domain.Agent{RevokedAt: &now}}

	_, err := New(repo).Handle(context.Background(), "dev")

	if !errors.Is(err, domain.ErrAgentRevoked) || repo.hash != nil {
		t.Errorf("error = %v", err)
	}
}

func TestHandleMissing(t *testing.T) {
	_, err := New(&fakeRepo{err: domain.ErrAgentNotFound}).Handle(context.Background(), "x")

	if !errors.Is(err, domain.ErrAgentNotFound) {
		t.Errorf("error = %v", err)
	}
}
