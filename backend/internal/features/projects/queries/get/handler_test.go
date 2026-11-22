package get

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

type fakeReader struct {
	project domain.Project
	roles   map[uuid.UUID]domain.Role
	asked   int
}

func (r *fakeReader) Get(_ context.Context, id uuid.UUID) (domain.Project, error) {
	if id != r.project.ID {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return r.project, nil
}

func (r *fakeReader) Member(_ context.Context, projectID, userID uuid.UUID) (domain.Member, error) {
	r.asked++
	role, ok := r.roles[userID]
	if !ok {
		return domain.Member{}, domain.ErrMemberNotFound
	}
	return domain.Member{ProjectID: projectID, UserID: userID, Role: role}, nil
}

var ana = kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1"), Channel: kernel.ChannelSession}

func TestHandleReturnsStatusAndProductVersion(t *testing.T) {
	project := domain.Project{ID: uuid.New(), Status: domain.StatusWorking, ProductVersion: "1.0.0"}

	result, err := New(&fakeReader{project: project}).Handle(context.Background(), ana, project.ID)

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusWorking || result.ProductVersion != "1.0.0" {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleReturnsActorRole(t *testing.T) {
	project := domain.Project{ID: uuid.New()}
	for role := range map[domain.Role]bool{domain.RoleOwner: true, domain.RoleEditor: true, domain.RoleViewer: true} {
		reader := &fakeReader{project: project, roles: map[uuid.UUID]domain.Role{ana.UserID: role}}

		result, err := New(reader).Handle(context.Background(), ana, project.ID)

		if err != nil || result.Role != role {
			t.Errorf("%s: result = %+v, err = %v", role, result, err)
		}
	}
}

func TestHandleWithoutUserOrMembershipHasNoRole(t *testing.T) {
	project := domain.Project{ID: uuid.New()}
	reader := &fakeReader{project: project}

	result, err := New(reader).Handle(context.Background(), kernel.Actor{}, project.ID)
	if err != nil || result.Role != "" || reader.asked != 0 {
		t.Errorf("sin usuario: result = %+v, consultas = %d, err = %v", result, reader.asked, err)
	}

	result, err = New(reader).Handle(context.Background(), ana, project.ID)
	if err != nil || result.Role != "" {
		t.Errorf("sin membresía: result = %+v, err = %v", result, err)
	}
}

func TestREST(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name   string
		id     string
		status int
	}{
		{"existe", id.String(), http.StatusOK},
		{"id inválido", "abc", http.StatusBadRequest},
		{"no existe", uuid.NewString(), http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{project: domain.Project{ID: id}}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+tt.id, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestRESTIncludesRole(t *testing.T) {
	id := uuid.New()
	mux := http.NewServeMux()
	New(&fakeReader{project: domain.Project{ID: id}, roles: map[uuid.UUID]domain.Role{ana.UserID: domain.RoleViewer}}).Register(mux)
	rec := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+id.String(), nil)
	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), ana)))

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"role":"viewer"`) {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}
