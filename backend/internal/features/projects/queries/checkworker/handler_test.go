package checkworker

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
)

var (
	project = uuid.MustParse("00000000-0000-0000-0000-000000000100")
	beaID   = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
)

type fakeMembers struct {
	role domain.Role
	err  error
}

func (m fakeMembers) Member(_ context.Context, projectID, userID uuid.UUID) (domain.Member, error) {
	if m.err != nil {
		return domain.Member{}, m.err
	}
	if m.role == "" || projectID != project || userID != beaID {
		return domain.Member{}, domain.ErrMemberNotFound
	}
	return domain.Member{ProjectID: projectID, UserID: userID, Role: m.role}, nil
}

type fakeUsers map[string]kernel.UserRef

func (u fakeUsers) FindByUsername(_ context.Context, username string) (kernel.UserRef, bool, error) {
	user, ok := u[username]
	return user, ok, nil
}

var users = fakeUsers{
	"bea":     {ID: beaID, Username: "bea", Active: true},
	"inactiv": {ID: uuid.New(), Username: "inactiv", Active: false},
}

func TestOwnerAndEditorWork(t *testing.T) {
	for _, role := range []domain.Role{domain.RoleOwner, domain.RoleEditor} {
		isUser, err := New(fakeMembers{role: role}, users).Handle(context.Background(), project, "bea")
		if !isUser || err != nil {
			t.Errorf("%s: usuario = %v, err = %v", role, isUser, err)
		}
	}
}

func TestViewerAndNonMemberConflict(t *testing.T) {
	for role, want := range map[domain.Role]error{domain.RoleViewer: domain.ErrWorkerIsViewer, "": domain.ErrWorkerNotMember} {
		isUser, err := New(fakeMembers{role: role}, users).Handle(context.Background(), project, "bea")
		if !isUser || !errors.Is(err, want) {
			t.Errorf("%q: usuario = %v, err = %v", role, isUser, err)
		}
		var business *kernel.Error
		if !errors.As(err, &business) || business.Kind != kernel.KindConflict {
			t.Errorf("%q: quiero conflict: %v", role, err)
		}
	}
}

func TestNotAUserIsLeftToAgents(t *testing.T) {
	for _, name := range []string{"dbp-tl", "inactiv"} {
		isUser, err := New(fakeMembers{role: domain.RoleEditor}, users).Handle(context.Background(), project, name)
		if isUser || err != nil {
			t.Errorf("%s: usuario = %v, err = %v", name, isUser, err)
		}
	}
}

func TestKeepsErrors(t *testing.T) {
	want := errors.New("sin base")
	if _, err := New(fakeMembers{err: want}, users).Handle(context.Background(), project, "bea"); !errors.Is(err, want) {
		t.Errorf("err = %v", err)
	}
}
