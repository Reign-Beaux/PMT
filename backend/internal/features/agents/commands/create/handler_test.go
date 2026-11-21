package create

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"pmt/internal/features/agents/domain"
	"pmt/internal/kernel"
)

type fakeRepo struct {
	hash []byte
	err  error
}

func (r *fakeRepo) Create(_ context.Context, agent domain.Agent, tokenHash []byte) (domain.Agent, error) {
	r.hash = tokenHash
	return agent, r.err
}

type noUsers struct{}

func (noUsers) Exists(context.Context, string) (bool, error) { return false, nil }

type users map[string]bool

func (u users) Exists(_ context.Context, name string) (bool, error) { return u[name], nil }

func TestHandleRejectsUsername(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, users{"saul": true}).Handle(context.Background(), Command{Name: "saul", Profile: kernel.ProfileTL})

	if !errors.Is(err, domain.ErrAgentNameIsUser) || repo.hash != nil {
		t.Errorf("error = %v, hash = %v", err, repo.hash)
	}
}

func TestHandleStoresOnlyTheHash(t *testing.T) {
	repo := &fakeRepo{}

	result, err := New(repo, noUsers{}).Handle(context.Background(), Command{Name: "dev", Profile: kernel.ProfileDevBackend})

	if err != nil {
		t.Fatal(err)
	}
	if result.Token == "" || !bytes.Equal(repo.hash, domain.HashToken(result.Token)) {
		t.Error("debe guardar el hash del token devuelto")
	}
}

func TestHandleRejectsInvalidProfile(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, noUsers{}).Handle(context.Background(), Command{Name: "dev", Profile: "X"})

	if !errors.Is(err, domain.ErrAgentProfileInvalid) || repo.hash != nil {
		t.Errorf("error = %v", err)
	}
}

func TestHandlePropagatesTakenName(t *testing.T) {
	_, err := New(&fakeRepo{err: domain.ErrAgentNameTaken}, noUsers{}).Handle(context.Background(), Command{Name: "dev", Profile: kernel.ProfileTL})

	if !errors.Is(err, domain.ErrAgentNameTaken) {
		t.Errorf("error = %v", err)
	}
}
