package listproductversionchanges

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type fakeReader struct {
	changes []domain.ProductVersionChange
	err     error
}

func (r fakeReader) ListProductVersionChanges(context.Context, uuid.UUID) ([]domain.ProductVersionChange, error) {
	return r.changes, r.err
}

func TestHandleMapsChanges(t *testing.T) {
	id := uuid.New()
	change := domain.ProductVersionChange{
		ID: uuid.New(), ProjectID: id, From: "1.0.0", To: "1.1.0",
		ActorName: "pp", ActorProfile: "PP", Reason: "etapa nueva", ChangedAt: time.Now(),
	}

	items, err := New(fakeReader{changes: []domain.ProductVersionChange{change}}).Handle(context.Background(), id)

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	got := items[0]
	if got.ProjectID != id || got.FromVersion != "1.0.0" || got.ToVersion != "1.1.0" || got.ActorName != "pp" || got.ActorProfile != "PP" || got.Reason != "etapa nueva" {
		t.Errorf("item = %+v", got)
	}
}

func TestHandleEmptyIsNotNil(t *testing.T) {
	items, err := New(fakeReader{}).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if items == nil || len(items) != 0 {
		t.Errorf("items = %#v", items)
	}
}

func TestHandlePropagatesError(t *testing.T) {
	want := errors.New("fallo")

	if _, err := New(fakeReader{err: want}).Handle(context.Background(), uuid.New()); !errors.Is(err, want) {
		t.Errorf("error = %v", err)
	}
}

func TestRESTListsChanges(t *testing.T) {
	mux := http.NewServeMux()
	New(fakeReader{}).Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+uuid.NewString()+"/product-version-changes", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body []Item
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body == nil {
		t.Errorf("cuerpo = %q, %v", rec.Body.String(), err)
	}
}

func TestRESTRejectsBadID(t *testing.T) {
	mux := http.NewServeMux()
	New(fakeReader{}).Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/x/product-version-changes", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

// TestToolAvailableToEveryProfile comprueba que la lectura no exige perfil: ni PC la tiene vedada.
func TestToolAvailableToEveryProfile(t *testing.T) {
	ctx := context.Background()
	server := platformmcp.NewServer()
	platformmcp.ActAs(server, kernel.Actor{Name: "pc", Profile: kernel.ProfilePC})
	New(fakeReader{}).RegisterTool(server)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "list_product_version_changes", Arguments: map[string]any{"project_id": uuid.NewString()}})

	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("resultado = %+v", result.Content[0])
	}
	var output toolOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &output); err != nil || output.Changes == nil || len(output.Changes) != 0 {
		t.Errorf("salida = %+v, %v", output, err)
	}
}
