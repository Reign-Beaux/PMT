package listmembers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

var (
	anaID = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	beaID = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
)

type fakeReader []domain.Member

func (r fakeReader) Members(context.Context, uuid.UUID) ([]domain.Member, error) {
	return r, nil
}

type fakeUsers map[uuid.UUID]kernel.UserRef

func (u fakeUsers) FindByUsername(context.Context, string) (kernel.UserRef, bool, error) {
	return kernel.UserRef{}, false, nil
}

func (u fakeUsers) FindByID(_ context.Context, id uuid.UUID) (kernel.UserRef, bool, error) {
	user, ok := u[id]
	return user, ok, nil
}

var (
	reader = fakeReader{{UserID: anaID, Role: domain.RoleOwner}, {UserID: beaID, Role: domain.RoleViewer}}
	users  = fakeUsers{anaID: {ID: anaID, Username: "ana", Name: "Ana"}, beaID: {ID: beaID, Username: "bea", Name: "Bea"}}
	want   = []domain.MemberItem{{Username: "ana", Name: "Ana", Role: domain.RoleOwner}, {Username: "bea", Name: "Bea", Role: domain.RoleViewer}}
)

func TestHandle(t *testing.T) {
	items, err := New(reader, users).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0] != want[0] || items[1] != want[1] {
		t.Errorf("items = %+v", items)
	}
}

func TestREST(t *testing.T) {
	mux := http.NewServeMux()
	New(reader, users).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+uuid.NewString()+"/members", nil))

	var got []domain.MemberItem
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusOK || len(got) != 2 || got[1] != want[1] {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestTool(t *testing.T) {
	ctx := context.Background()
	server := platformmcp.NewServer()
	New(reader, users).RegisterTool(server)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "list_project_members", Arguments: map[string]any{"project_id": uuid.NewString()}})
	if err != nil || result.IsError {
		t.Fatalf("resultado = %+v, err = %v", result, err)
	}
	var output toolOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Members) != 2 || output.Members[0] != (toolMember{Username: "ana", Name: "Ana", Role: "owner"}) {
		t.Errorf("miembros = %+v", output.Members)
	}
}
