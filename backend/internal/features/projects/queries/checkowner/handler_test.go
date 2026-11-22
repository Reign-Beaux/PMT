package checkowner

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

type fakeMembers struct {
	role domain.Role
	err  error
}

func (f fakeMembers) Member(context.Context, uuid.UUID, uuid.UUID) (domain.Member, error) {
	if f.err != nil {
		return domain.Member{}, f.err
	}
	return domain.Member{Role: f.role}, nil
}

func TestHandle(t *testing.T) {
	ctx := context.Background()
	project, user := uuid.New(), uuid.New()
	cases := []struct {
		name    string
		members fakeMembers
		user    uuid.UUID
		want    bool
	}{
		{"owner", fakeMembers{role: domain.RoleOwner}, user, true},
		{"editor", fakeMembers{role: domain.RoleEditor}, user, false},
		{"viewer", fakeMembers{role: domain.RoleViewer}, user, false},
		{"no miembro", fakeMembers{err: domain.ErrMemberNotFound}, user, false},
		{"sin usuario", fakeMembers{role: domain.RoleOwner}, uuid.Nil, false},
	}
	for _, c := range cases {
		got, err := New(c.members).Handle(ctx, project, c.user)
		if err != nil || got != c.want {
			t.Errorf("%s: owner = %v, err = %v", c.name, got, err)
		}
	}
}

func TestHandleKeepsErrors(t *testing.T) {
	want := errors.New("sin base")
	if _, err := New(fakeMembers{err: want}).Handle(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
}
