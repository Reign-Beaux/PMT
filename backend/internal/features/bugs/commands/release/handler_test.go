package release

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
	"pmt/internal/platform/rest"
)

var (
	ownerID = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	anaID   = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	owner   = kernel.Actor{Name: "saul", Profile: kernel.ProfileUser, UserID: ownerID}
)

type fakeRepo struct {
	bug      domain.Bug
	reads    int
	released bool
	holder   uuid.UUID
	change   domain.StatusChange
	err      error
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Bug, error) {
	r.reads++
	s := r.bug
	s.ID = id
	return s, nil
}

func (r *fakeRepo) Release(_ context.Context, s domain.Bug, holder uuid.UUID, change domain.StatusChange) (domain.Bug, error) {
	if r.err != nil {
		return domain.Bug{}, r.err
	}
	r.released, r.holder, r.change = true, holder, change
	return s, nil
}

// owners: solo ownerID es owner.
type owners struct{}

func (owners) IsOwner(_ context.Context, _ uuid.UUID, userID uuid.UUID) (bool, error) {
	return userID == ownerID, nil
}

func taken() domain.Bug {
	return domain.Bug{Status: domain.StatusInDev, Hold: kernel.Hold{HolderID: anaID, TakenBy: "ana"}}
}

func TestHandleReleases(t *testing.T) {
	for _, actor := range []kernel.Actor{owner, {Name: "saul-pp", Profile: kernel.ProfilePP, UserID: ownerID}} {
		repo := &fakeRepo{bug: taken()}

		result, err := New(repo, owners{}).Handle(context.Background(), Command{ID: uuid.New(), Reason: "ana se fue", Actor: actor})

		if err != nil {
			t.Fatalf("%s: %v", actor.Profile, err)
		}
		if !repo.released || repo.holder != anaID || result.Taken() || result.TakenBy != "" {
			t.Fatalf("%s: liberada = %v, quien = %v, resultado = %+v", actor.Profile, repo.released, repo.holder, result)
		}
		if repo.change.From != domain.StatusInDev || repo.change.To != domain.StatusInDev || !strings.HasSuffix(repo.change.Reason, "ana se fue") {
			t.Errorf("registro = %+v", repo.change)
		}
	}
}

func TestHandleRejects(t *testing.T) {
	editor := kernel.Actor{Name: "bea", Profile: kernel.ProfileUser, UserID: uuid.New()}
	tests := []struct {
		name   string
		actor  kernel.Actor
		bug    domain.Bug
		reason string
		want   error
	}{
		{"no es el owner", editor, taken(), "motivo", domain.ErrReleaseNotOwner},
		{"no está tomada", owner, domain.Bug{Status: domain.StatusInDev}, "motivo", domain.ErrBugNotTaken},
		{"sin motivo", owner, taken(), " ", domain.ErrReleaseReasonRequired},
	}
	for _, tt := range tests {
		repo := &fakeRepo{bug: tt.bug}

		_, err := New(repo, owners{}).Handle(context.Background(), Command{ID: uuid.New(), Reason: tt.reason, Actor: tt.actor})

		if !errors.Is(err, tt.want) || repo.released {
			t.Errorf("%s: err = %v, liberada = %v", tt.name, err, repo.released)
		}
	}
}

func TestHandleRejectsProfileBeforeReading(t *testing.T) {
	for _, profile := range []kernel.Profile{kernel.ProfileTL, kernel.ProfilePC, kernel.ProfileDP, kernel.ProfileDevBackend, kernel.ProfileQA, ""} {
		repo := &fakeRepo{bug: taken()}

		_, err := New(repo, owners{}).Handle(context.Background(), Command{ID: uuid.New(), Reason: "motivo", Actor: kernel.Actor{Name: "x", Profile: profile, UserID: ownerID}})

		if !errors.Is(err, domain.ErrReleaseForbidden) || repo.reads != 0 {
			t.Errorf("%s: err = %v, lecturas = %d", profile, err, repo.reads)
		}
	}
}

func TestHandleKeepsRepositoryErrors(t *testing.T) {
	repo := &fakeRepo{bug: taken(), err: domain.ErrBugHoldChanged}

	if _, err := New(repo, owners{}).Handle(context.Background(), Command{ID: uuid.New(), Reason: "motivo", Actor: owner}); !errors.Is(err, domain.ErrBugHoldChanged) {
		t.Fatalf("err = %v", err)
	}
}

func TestRESTReleases(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeRepo{bug: taken()}, owners{}).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/bugs/"+uuid.NewString()+"/release", strings.NewReader(`{"reason":"ana se fue"}`))
	req = req.WithContext(rest.WithActor(req.Context(), owner))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusOK || body["taken_by"] != nil || body["status"] != "in_dev" {
		t.Fatalf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestToolReleases(t *testing.T) {
	ctx := context.Background()
	server := platformmcp.NewServer()
	New(&fakeRepo{bug: taken()}, owners{}).RegisterTool(server)
	platformmcp.ActAs(server, owner)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "release_bug", Arguments: map[string]any{"bug_id": uuid.NewString(), "reason": "ana se fue"}})

	if err != nil || result.IsError {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
	result, _ = session.CallTool(ctx, &sdk.CallToolParams{Name: "release_bug", Arguments: map[string]any{"bug_id": "x", "reason": "r"}})
	if !result.IsError {
		t.Fatal("un ID inválido debe fallar")
	}
}
