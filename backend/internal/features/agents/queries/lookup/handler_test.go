package lookup

import (
	"context"
	"errors"
	"testing"
	"time"

	"pmt/internal/features/agents/domain"
	"pmt/internal/kernel"
)

type fakeReader struct {
	agent domain.Agent
	err   error
}

func (f fakeReader) GetByName(context.Context, string) (domain.Agent, error) {
	return f.agent, f.err
}

func TestHandle(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		reader  fakeReader
		want    kernel.Profile
		wantErr error
	}{
		{"activo", fakeReader{agent: domain.Agent{Profile: kernel.ProfileTL}}, kernel.ProfileTL, nil},
		{"revocado", fakeReader{agent: domain.Agent{Profile: kernel.ProfileTL, RevokedAt: &now}}, "", domain.ErrAgentRevoked},
		{"inexistente", fakeReader{err: domain.ErrAgentNotFound}, "", domain.ErrAgentNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.reader).Handle(context.Background(), "a")
			if got != tt.want || !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}
