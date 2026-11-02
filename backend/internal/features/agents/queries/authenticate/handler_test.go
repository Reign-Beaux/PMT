package authenticate

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"pmt/internal/features/agents/domain"
	"pmt/internal/kernel"
)

type fakeReader struct {
	hash []byte
}

func (r fakeReader) GetActiveByTokenHash(_ context.Context, tokenHash []byte) (domain.Agent, error) {
	if !bytes.Equal(tokenHash, r.hash) {
		return domain.Agent{}, kernel.ErrUnauthenticated
	}
	return domain.Agent{Name: "tl", Profile: kernel.ProfileTL}, nil
}

func TestAuthenticate(t *testing.T) {
	handler := New(fakeReader{hash: domain.HashToken("bueno")})

	actor, err := handler.Authenticate(context.Background(), "bueno")

	if err != nil {
		t.Fatal(err)
	}
	if actor != (kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}) {
		t.Errorf("actor = %+v", actor)
	}
}

func TestAuthenticateRejectsUnknownToken(t *testing.T) {
	handler := New(fakeReader{hash: domain.HashToken("bueno")})

	if _, err := handler.Authenticate(context.Background(), "malo"); !errors.Is(err, kernel.ErrUnauthenticated) {
		t.Errorf("error = %v", err)
	}
}
