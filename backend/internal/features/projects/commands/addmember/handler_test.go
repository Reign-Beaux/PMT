package addmember

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
	"pmt/internal/platform/rest"
)

var (
	ownerID  = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	editorID = uuid.MustParse("00000000-0000-0000-0000-0000000000a2")
	beaID    = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	owner    = kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: ownerID}
)

type fakeRepo struct {
	roles map[uuid.UUID]domain.Role
	added []domain.Member
	err   error
}

func newRepo() *fakeRepo {
	return &fakeRepo{roles: map[uuid.UUID]domain.Role{ownerID: domain.RoleOwner, editorID: domain.RoleEditor}}
}

func (r *fakeRepo) Member(_ context.Context, projectID, userID uuid.UUID) (domain.Member, error) {
	role, ok := r.roles[userID]
	if !ok {
		return domain.Member{}, domain.ErrMemberNotFound
	}
	return domain.Member{ProjectID: projectID, UserID: userID, Role: role}, nil
}

func (r *fakeRepo) AddMember(_ context.Context, m domain.Member) error {
	if r.err != nil {
		return r.err
	}
	r.added = append(r.added, m)
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
	"bea":  {ID: beaID, Username: "bea", Name: "Bea", Active: true},
	"baja": {ID: uuid.New(), Username: "baja", Name: "Baja", Active: false},
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name     string
		actor    kernel.Actor
		username string
		role     string
		repoErr  error
		want     error
	}{
		{"owner agrega editor", owner, "bea", "editor", nil, nil},
		{"owner agrega viewer", owner, " bea ", "viewer", nil, nil},
		{"owner con perfil declarado", kernel.Actor{Name: "ana", Profile: kernel.ProfilePP, UserID: ownerID}, "bea", "editor", nil, domain.ErrMembersForbidden},
		{"editor", kernel.Actor{Name: "eli", Profile: kernel.ProfileUser, UserID: editorID}, "bea", "editor", nil, domain.ErrNotProjectOwner},
		{"no miembro", kernel.Actor{Name: "zoe", Profile: kernel.ProfileUser, UserID: uuid.New()}, "bea", "editor", nil, domain.ErrProjectNotFound},
		{"rol owner", owner, "bea", "owner", nil, domain.ErrRoleInvalid},
		{"sin username", owner, " ", "editor", nil, domain.ErrUsernameRequired},
		{"usuario inexistente", owner, "nadie", "editor", nil, domain.ErrMemberUserNotFound},
		{"usuario desactivado", owner, "baja", "viewer", nil, domain.ErrMemberUserNotFound},
		{"ya es miembro", owner, "bea", "editor", domain.ErrMemberExists, domain.ErrMemberExists},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo()
			repo.err = tt.repoErr

			item, err := New(repo, users).Handle(context.Background(), Command{ProjectID: uuid.New(), Username: tt.username, Role: tt.role, Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if tt.want == nil && (len(repo.added) != 1 || repo.added[0].UserID != beaID || item.Username != "bea" || string(item.Role) != tt.role) {
				t.Errorf("agregados = %+v, item = %+v", repo.added, item)
			}
			if tt.want != nil && tt.repoErr == nil && len(repo.added) != 0 {
				t.Error("no debió agregar")
			}
		})
	}
}

func TestREST(t *testing.T) {
	mux := http.NewServeMux()
	repo := newRepo()
	New(repo, users).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+uuid.NewString()+"/members", strings.NewReader(`{"username":"bea","role":"viewer"}`))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), owner)))

	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"role":"viewer"`) {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestTool(t *testing.T) {
	ctx := context.Background()
	server := platformmcp.NewServer()
	platformmcp.ActAs(server, owner)
	New(newRepo(), users).RegisterTool(server)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "add_project_member", Arguments: map[string]any{"project_id": uuid.NewString(), "username": "bea", "role": "admin"}})
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.HasPrefix(text, "invalid: rol inválido") {
		t.Errorf("resultado = %q", text)
	}
}
