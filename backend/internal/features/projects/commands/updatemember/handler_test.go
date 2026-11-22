package updatemember

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
	updated []domain.Member
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

func (r *fakeRepo) UpdateMemberRole(_ context.Context, m domain.Member) error {
	r.updated = append(r.updated, m)
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

var users = fakeUsers{
	"ana":  {ID: ownerID, Username: "ana", Active: true},
	"bea":  {ID: beaID, Username: "bea", Name: "Bea", Active: true},
	"otro": {ID: uuid.New(), Username: "otro", Active: true},
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name     string
		actor    kernel.Actor
		username string
		role     string
		want     error
	}{
		{"viewer a editor", owner, "bea", "editor", nil},
		{"perfil declarado", kernel.Actor{Name: "ana", Profile: kernel.ProfileTL, UserID: ownerID}, "bea", "editor", domain.ErrMembersForbidden},
		{"editor", kernel.Actor{Name: "eli", Profile: kernel.ProfileUser, UserID: editorID}, "bea", "editor", domain.ErrNotProjectOwner},
		{"no miembro", kernel.Actor{Name: "zoe", Profile: kernel.ProfileUser, UserID: uuid.New()}, "bea", "editor", domain.ErrProjectNotFound},
		{"rol inválido", owner, "bea", "owner", domain.ErrRoleInvalid},
		{"al owner", owner, "ana", "viewer", domain.ErrOwnerUnchangeable},
		{"usuario que no es miembro", owner, "otro", "viewer", domain.ErrMemberNotFound},
		{"usuario inexistente", owner, "nadie", "viewer", domain.ErrMemberNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo()

			item, err := New(repo, users).Handle(context.Background(), Command{ProjectID: uuid.New(), Username: tt.username, Role: tt.role, Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if tt.want == nil && (len(repo.updated) != 1 || repo.updated[0].Role != domain.RoleEditor || item.Role != domain.RoleEditor) {
				t.Errorf("cambios = %+v, item = %+v", repo.updated, item)
			}
			if tt.want != nil && len(repo.updated) != 0 {
				t.Error("no debió cambiar")
			}
		})
	}
}

func TestREST(t *testing.T) {
	mux := http.NewServeMux()
	New(newRepo(), users).Register(mux)
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+uuid.NewString()+"/members/bea", strings.NewReader(`{"role":"editor"}`))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), owner)))

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"username":"bea"`) {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}
