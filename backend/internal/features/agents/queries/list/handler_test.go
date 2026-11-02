package list

import (
	"context"
	"testing"
	"time"

	"pmt/internal/features/agents/domain"
	"pmt/internal/kernel"
)

type fakeReader []domain.Agent

func (r fakeReader) List(context.Context) ([]domain.Agent, error) {
	return r, nil
}

func TestHandle(t *testing.T) {
	now := time.Now()
	reader := fakeReader{{Name: "a", Profile: kernel.ProfileTL}, {Name: "b", Profile: kernel.ProfileQA, RevokedAt: &now}}

	items, err := New(reader).Handle(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Revoked || !items[1].Revoked || items[1].Profile != kernel.ProfileQA {
		t.Errorf("items = %+v", items)
	}
}
