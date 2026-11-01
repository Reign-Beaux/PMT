package add

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/activity/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

var (
	errProjectNotFound = kernel.NotFound("proyecto no encontrado")
	errStageNotFound   = kernel.NotFound("etapa no encontrada")
	errTaskNotFound    = kernel.NotFound("tarea no encontrada")
	errBugNotFound     = kernel.NotFound("bug no encontrado")
	tl                 = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
)

type fakeRepo struct {
	created []domain.Comment
}

func (r *fakeRepo) Create(_ context.Context, c domain.Comment) (domain.Comment, error) {
	r.created = append(r.created, c)
	return c, nil
}

// fakeItem confirma solo el ID existente, y responde su error con cualquier otro.
type fakeItem struct {
	existing uuid.UUID
	missing  error
}

func (f fakeItem) Check(_ context.Context, id uuid.UUID) error {
	if id != f.existing {
		return f.missing
	}
	return nil
}

func (f fakeItem) StageOf(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), f.Check(ctx, id)
}

func newHandler(repo *fakeRepo, existing uuid.UUID) *Handler {
	return New(repo,
		fakeItem{existing, errProjectNotFound},
		fakeItem{existing, errStageNotFound},
		fakeItem{existing, errTaskNotFound},
		fakeItem{existing, errBugNotFound})
}

func TestHandleAdds(t *testing.T) {
	id := uuid.New()
	actors := []kernel.Actor{
		tl,
		kernel.Owner,
		{Name: "pc", Profile: kernel.ProfilePC},
		{Name: "dev", Profile: kernel.ProfileDevBackend},
	}
	for _, kind := range []domain.ItemKind{domain.ItemProject, domain.ItemStage, domain.ItemTask, domain.ItemBug} {
		for _, actor := range actors {
			t.Run(string(kind)+" "+actor.Name, func(t *testing.T) {
				repo := &fakeRepo{}

				result, err := newHandler(repo, id).Handle(context.Background(), Command{ItemKind: kind, ItemID: id, Body: " Va a QA ", Actor: actor})

				if err != nil {
					t.Fatal(err)
				}
				if len(repo.created) != 1 {
					t.Fatalf("creados = %d", len(repo.created))
				}
				if result.ItemKind != kind || result.ItemID != id || result.Body != "Va a QA" || result.ActorName != actor.Name || result.ActorProfile != actor.Profile {
					t.Errorf("resultado = %+v", result)
				}
			})
		}
	}
}

func TestHandleRejects(t *testing.T) {
	id := uuid.New()
	other := uuid.New()
	tests := []struct {
		name string
		cmd  Command
		want error
	}{
		{"vacío", Command{ItemKind: domain.ItemTask, ItemID: id}, domain.ErrBodyRequired},
		{"solo espacios", Command{ItemKind: domain.ItemTask, ItemID: id, Body: "  "}, domain.ErrBodyRequired},
		{"tipo desconocido", Command{ItemKind: "finding", ItemID: id, Body: "Texto"}, domain.ErrItemKindInvalid},
		{"proyecto inexistente", Command{ItemKind: domain.ItemProject, ItemID: other, Body: "Texto"}, errProjectNotFound},
		{"etapa inexistente", Command{ItemKind: domain.ItemStage, ItemID: other, Body: "Texto"}, errStageNotFound},
		{"tarea inexistente", Command{ItemKind: domain.ItemTask, ItemID: other, Body: "Texto"}, errTaskNotFound},
		{"bug inexistente", Command{ItemKind: domain.ItemBug, ItemID: other, Body: "Texto"}, errBugNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			tt.cmd.Actor = tl

			_, err := newHandler(repo, id).Handle(context.Background(), tt.cmd)

			if !errors.Is(err, tt.want) {
				t.Errorf("error = %v", err)
			}
			if len(repo.created) != 0 {
				t.Errorf("creados = %d", len(repo.created))
			}
		})
	}
}

func TestREST(t *testing.T) {
	id := uuid.New()
	body := func(kind, item, text string) string {
		return `{"item_kind":"` + kind + `","item_id":"` + item + `","body":"` + text + `"}`
	}
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{"válido", body("task", id.String(), "Texto"), http.StatusCreated},
		{"sin texto", body("task", id.String(), " "), http.StatusBadRequest},
		{"tipo desconocido", body("finding", id.String(), "Texto"), http.StatusBadRequest},
		{"id inválido", body("task", "abc", "Texto"), http.StatusBadRequest},
		{"no existe", body("bug", uuid.NewString(), "Texto"), http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			newHandler(&fakeRepo{}, id).Register(mux)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/comments", strings.NewReader(tt.body))

			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), kernel.Owner)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestRESTTakesActorFromToken(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{}
	mux := http.NewServeMux()
	newHandler(repo, id).Register(mux)
	rec := httptest.NewRecorder()
	raw := `{"item_kind":"task","item_id":"` + id.String() + `","body":"Texto"}`
	req := httptest.NewRequest(http.MethodPost, "/api/comments", strings.NewReader(raw))

	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), tl)))

	if len(repo.created) != 1 || repo.created[0].ActorName != "tl" || repo.created[0].ActorProfile != kernel.ProfileTL {
		t.Fatalf("creados = %+v, cuerpo = %s", repo.created, rec.Body)
	}
}
