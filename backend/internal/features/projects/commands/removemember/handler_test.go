package removemember

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

var (
	ownerID  = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	editorID = uuid.MustParse("00000000-0000-0000-0000-0000000000a2")
	beaID    = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	owner    = kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: ownerID}
)

type fakeRepo struct {
	roles   map[uuid.UUID]domain.Role
	removed []uuid.UUID
}

func newRepo() *fakeRepo {
	return &fakeRepo{roles: map[uuid.UUID]domain.Role{ownerID: domain.RoleOwner, editorID: domain.RoleEditor, beaID: domain.RoleViewer}}
}

func (r *fakeRepo) Member(_ context.Context, projectID, userID uuid.UUID) (domain.Member, error) {
	role, ok := r.roles[userID]
	if !ok {
		return domain.Member{}, domain.ErrMemberNotFound
	}
	return domain.Member{ProjectID: projectID, UserID: userID, Role: role}, nil
}

func (r *fakeRepo) RemoveMember(_ context.Context, _, userID uuid.UUID) error {
	r.removed = append(r.removed, userID)
	return nil
}

type fakeUsers map[string]kernel.UserRef

func (u fakeUsers) FindByUsername(_ context.Context, username string) (kernel.UserRef, bool, error) {
	user, ok := u[username]
	return user, ok, nil
}

func (u fakeUsers) FindByID(context.Context, uuid.UUID) (kernel.UserRef, bool, error) {
	return kernel.UserRef{}, false, nil
}

// La baja está desactivada: conserva su membresía y el owner la puede quitar.
var users = fakeUsers{
	"ana": {ID: ownerID, Username: "ana", Active: true},
	"bea": {ID: beaID, Username: "bea", Active: false},
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name     string
		actor    kernel.Actor
		username string
		want     error
	}{
		{"quita a un usuario desactivado", owner, "bea", nil},
		{"perfil declarado", kernel.Actor{Name: "ana", Profile: kernel.ProfilePP, UserID: ownerID}, "bea", domain.ErrMembersForbidden},
		{"editor", kernel.Actor{Name: "eli", Profile: kernel.ProfileUser, UserID: editorID}, "bea", domain.ErrNotProjectOwner},
		{"no miembro", kernel.Actor{Name: "zoe", Profile: kernel.ProfileUser, UserID: uuid.New()}, "bea", domain.ErrProjectNotFound},
		{"al owner", owner, "ana", domain.ErrOwnerUnchangeable},
		{"usuario inexistente", owner, "nadie", domain.ErrMemberNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo()

			err := New(repo, users).Handle(context.Background(), Command{ProjectID: uuid.New(), Username: tt.username, Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if (tt.want == nil) != (len(repo.removed) == 1) {
				t.Errorf("quitados = %v", repo.removed)
			}
		})
	}
}

func TestREST(t *testing.T) {
	mux := http.NewServeMux()
	New(newRepo(), users).Register(mux)
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+uuid.NewString()+"/members/bea", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), owner)))

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}
