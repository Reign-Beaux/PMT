package list

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

type fakeReader struct {
	projects []domain.MemberProject
	users    []uuid.UUID
	roles    []domain.Role
}

func (r *fakeReader) List(_ context.Context, userID uuid.UUID, roles []domain.Role) ([]domain.MemberProject, error) {
	r.users = append(r.users, userID)
	r.roles = roles
	return r.projects, nil
}

var ana = kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1")}

func TestHandleListsActorProjects(t *testing.T) {
	reader := &fakeReader{projects: []domain.MemberProject{{Project: domain.Project{Name: "A"}}}}

	items, err := New(reader).Handle(context.Background(), ana)

	if err != nil || len(items) != 1 {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
	if len(reader.users) != 1 || reader.users[0] != ana.UserID {
		t.Errorf("consultó por %v", reader.users)
	}
}

func TestHandleRolesByChannel(t *testing.T) {
	for channel, want := range map[kernel.Channel][]domain.Role{
		kernel.ChannelToken:   {domain.RoleOwner, domain.RoleEditor},
		kernel.ChannelSession: {domain.RoleOwner, domain.RoleEditor, domain.RoleViewer},
	} {
		reader := &fakeReader{}
		actor := ana
		actor.Channel = channel

		if _, err := New(reader).Handle(context.Background(), actor); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(reader.roles, want) {
			t.Errorf("%s: roles = %v, quiero %v", channel, reader.roles, want)
		}
	}
}

func TestHandleWithoutUserSeesNothing(t *testing.T) {
	reader := &fakeReader{projects: []domain.MemberProject{{Project: domain.Project{Name: "A"}}}}

	items, err := New(reader).Handle(context.Background(), kernel.Actor{Name: "dbp-tl", Profile: kernel.ProfileTL})

	if err != nil || len(items) != 0 || len(reader.users) != 0 {
		t.Fatalf("items = %+v, consultas = %d, err = %v", items, len(reader.users), err)
	}
}

func TestRESTReturnsEmptyArray(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}).Register(mux)
	rec := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), ana)))

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestRESTIncludesRole(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{projects: []domain.MemberProject{{Project: domain.Project{Name: "A"}, Role: domain.RoleEditor}}}).Register(mux)
	rec := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), ana)))

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"role":"editor"`) {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestHandleReturnsProjects(t *testing.T) {
	reader := &fakeReader{projects: []domain.MemberProject{
		{Project: domain.Project{Name: "A"}, Role: domain.RoleOwner},
		{Project: domain.Project{Name: "B", Status: domain.StatusInDev, ProductVersion: "1.0.0"}, Role: domain.RoleViewer},
	}}

	items, err := New(reader).Handle(context.Background(), ana)

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Name != "B" || items[1].Status != domain.StatusInDev || items[1].ProductVersion != "1.0.0" || items[0].Role != domain.RoleOwner || items[1].Role != domain.RoleViewer {
		t.Errorf("items = %+v", items)
	}
}
