package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

func TestRoleCanWrite(t *testing.T) {
	for role, want := range map[Role]bool{RoleOwner: true, RoleEditor: true, RoleViewer: false, "": false} {
		if role.CanWrite() != want {
			t.Errorf("%q escribe = %v", role, !want)
		}
	}
}

func TestRoleReaches(t *testing.T) {
	web := kernel.Actor{Channel: kernel.ChannelSession}
	for _, actor := range []kernel.Actor{{Channel: kernel.ChannelToken}, {}} {
		for role, want := range map[Role]bool{RoleOwner: true, RoleEditor: true, RoleViewer: false, "": false} {
			if role.Reaches(actor) != want {
				t.Errorf("token %q: %q llega = %v", actor.Channel, role, !want)
			}
		}
	}
	for role, want := range map[Role]bool{RoleOwner: true, RoleEditor: true, RoleViewer: true, "": false} {
		if role.Reaches(web) != want {
			t.Errorf("cookie: %q llega = %v", role, !want)
		}
	}
}

func TestReachableRoles(t *testing.T) {
	if got := ReachableRoles(kernel.Actor{Channel: kernel.ChannelToken}); len(got) != 2 || got[0] != RoleOwner || got[1] != RoleEditor {
		t.Errorf("token: %v", got)
	}
	if got := ReachableRoles(kernel.Actor{}); len(got) != 2 {
		t.Errorf("sin canal cuenta como token: %v", got)
	}
	if got := ReachableRoles(kernel.Actor{Channel: kernel.ChannelSession}); len(got) != 3 {
		t.Errorf("cookie: %v", got)
	}
}

func TestEnsureWorks(t *testing.T) {
	for role, want := range map[Role]error{RoleOwner: nil, RoleEditor: nil, RoleViewer: ErrWorkerIsViewer, "": ErrWorkerNotMember} {
		if err := EnsureWorks(role); !errors.Is(err, want) && err != want {
			t.Errorf("%q: err = %v", role, err)
		}
	}
}

func TestParseMemberRole(t *testing.T) {
	for _, value := range []string{"editor", " viewer "} {
		if _, err := ParseMemberRole(value); err != nil {
			t.Errorf("%q: %v", value, err)
		}
	}
	for _, value := range []string{"owner", "", "Editor", "admin"} {
		if _, err := ParseMemberRole(value); !errors.Is(err, ErrRoleInvalid) {
			t.Errorf("%q: err = %v", value, err)
		}
	}
}

func TestEnsureCanManageMembers(t *testing.T) {
	if err := EnsureCanManageMembers(kernel.Actor{Name: "saul", Profile: kernel.ProfileUser}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []kernel.Profile{kernel.ProfilePP, kernel.ProfileTL, kernel.ProfilePC, ""} {
		if err := EnsureCanManageMembers(kernel.Actor{Name: "saul", Profile: p}); !errors.Is(err, ErrMembersForbidden) {
			t.Errorf("%q: err = %v", p, err)
		}
	}
}

func TestEnsureManages(t *testing.T) {
	if err := EnsureManages(RoleOwner); err != nil {
		t.Fatal(err)
	}
	for _, role := range []Role{RoleEditor, RoleViewer} {
		if err := EnsureManages(role); !errors.Is(err, ErrNotProjectOwner) {
			t.Errorf("%s: err = %v", role, err)
		}
	}
}

func TestEnsureChangeable(t *testing.T) {
	if err := (Member{Role: RoleOwner}).EnsureChangeable(); !errors.Is(err, ErrOwnerUnchangeable) {
		t.Errorf("owner: err = %v", err)
	}
	for _, role := range []Role{RoleEditor, RoleViewer} {
		if err := (Member{Role: role}).EnsureChangeable(); err != nil {
			t.Errorf("%s: %v", role, err)
		}
	}
}

func TestOwnerOf(t *testing.T) {
	id := uuid.New()
	if got, err := OwnerOf(kernel.Actor{UserID: id}); err != nil || got != id {
		t.Fatalf("owner = %v, err = %v", got, err)
	}
	if _, err := OwnerOf(kernel.Actor{Name: "dbp-tl"}); !errors.Is(err, ErrNoUser) {
		t.Errorf("err = %v", err)
	}
}

type fakeMembers map[uuid.UUID]Role

func (f fakeMembers) Member(_ context.Context, projectID, userID uuid.UUID) (Member, error) {
	role, ok := f[userID]
	if !ok {
		return Member{}, ErrMemberNotFound
	}
	return Member{ProjectID: projectID, UserID: userID, Role: role}, nil
}

func TestEnsureOwner(t *testing.T) {
	owner, editor, stranger := uuid.New(), uuid.New(), uuid.New()
	members := fakeMembers{owner: RoleOwner, editor: RoleEditor}
	tests := []struct {
		name  string
		actor kernel.Actor
		want  error
	}{
		{"owner", kernel.Actor{UserID: owner}, nil},
		{"editor", kernel.Actor{UserID: editor}, ErrNotProjectOwner},
		{"no miembro", kernel.Actor{UserID: stranger}, ErrProjectNotFound},
		{"sin usuario", kernel.Actor{Name: "dbp-tl"}, ErrProjectNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := EnsureOwner(context.Background(), members, uuid.New(), tt.actor); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, quiero %v", err, tt.want)
			}
		})
	}
}
