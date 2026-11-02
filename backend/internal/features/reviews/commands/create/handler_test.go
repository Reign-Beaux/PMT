package create

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/reviews/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

var (
	errTaskNotFound = kernel.NotFound("tarea no encontrada")
	errBugNotFound  = kernel.NotFound("bug no encontrado")
	cr              = kernel.Actor{Name: "cr", Profile: kernel.ProfileCR}
	qa              = kernel.Actor{Name: "qa", Profile: kernel.ProfileQA}
	blocking        = true
)

type fakeRepo struct {
	created []domain.Finding
}

func (r *fakeRepo) Create(_ context.Context, f domain.Finding) (domain.Finding, error) {
	r.created = append(r.created, f)
	return f, nil
}

type fakeItems struct {
	existing uuid.UUID
	calls    int
}

func (f *fakeItems) StageOf(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
	f.calls++
	if id != f.existing {
		return uuid.Nil, errTaskNotFound
	}
	return uuid.New(), nil
}

func (f *fakeItems) Check(_ context.Context, id uuid.UUID) error {
	f.calls++
	if id != f.existing {
		return errBugNotFound
	}
	return nil
}

func TestHandleCreates(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name  string
		kind  domain.ItemKind
		actor kernel.Actor
	}{
		{"CR en tarea", domain.ItemTask, cr},
		{"QA en tarea", domain.ItemTask, qa},
		{"CR en bug", domain.ItemBug, cr},
		{"QA en bug", domain.ItemBug, qa},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			items := &fakeItems{existing: id}

			result, err := New(repo, items, items).Handle(context.Background(), Command{ItemKind: tt.kind, ItemID: id, Description: "Falla", Blocking: &blocking, Actor: tt.actor})

			if err != nil {
				t.Fatal(err)
			}
			if result.Status != domain.StatusOpen || !result.Blocking || result.ActorName != tt.actor.Name || len(repo.created) != 1 {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func TestHandleRejects(t *testing.T) {
	id := uuid.New()
	valid := Command{ItemKind: domain.ItemTask, ItemID: id, Description: "Falla", Blocking: &blocking, Actor: cr}
	with := func(change func(*Command)) Command {
		cmd := valid
		change(&cmd)
		return cmd
	}
	tests := []struct {
		name  string
		cmd   Command
		want  error
		calls int
	}{
		{"TL", with(func(c *Command) { c.Actor = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL} }), domain.ErrCreateForbidden, 0},
		{"Dev", with(func(c *Command) { c.Actor = kernel.Actor{Name: "dev", Profile: kernel.ProfileDevBackend} }), domain.ErrCreateForbidden, 0},
		{"User", with(func(c *Command) { c.Actor = kernel.Owner }), domain.ErrCreateForbidden, 0},
		{"tipo desconocido", with(func(c *Command) { c.ItemKind = "stage" }), domain.ErrItemKindInvalid, 0},
		{"sin blocking", with(func(c *Command) { c.Blocking = nil }), domain.ErrBlockingRequired, 0},
		{"sin descripción", with(func(c *Command) { c.Description = "  " }), domain.ErrDescriptionRequired, 0},
		{"tarea inexistente", with(func(c *Command) { c.ItemID = uuid.New() }), errTaskNotFound, 1},
		{"bug inexistente", with(func(c *Command) { c.ItemKind = domain.ItemBug; c.ItemID = uuid.New() }), errBugNotFound, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			items := &fakeItems{existing: id}

			_, err := New(repo, items, items).Handle(context.Background(), tt.cmd)

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if len(repo.created) != 0 || items.calls != tt.calls {
				t.Errorf("creados = %d, consultas = %d", len(repo.created), items.calls)
			}
		})
	}
}

func TestREST(t *testing.T) {
	id := uuid.New()
	body := func(item string) string {
		return `{"item_kind":"task","item_id":"` + item + `","description":"Falla","blocking":false}`
	}
	tests := []struct {
		name   string
		body   string
		actor  kernel.Actor
		status int
	}{
		{"válido", body(id.String()), cr, http.StatusCreated},
		{"perfil sin permiso", body(id.String()), kernel.Owner, http.StatusForbidden},
		{"sin blocking", `{"item_kind":"task","item_id":"` + id.String() + `","description":"Falla"}`, cr, http.StatusBadRequest},
		{"id inválido", body("abc"), cr, http.StatusBadRequest},
		{"no existe", body(uuid.NewString()), cr, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			items := &fakeItems{existing: id}
			New(&fakeRepo{}, items, items).Register(mux)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/findings", strings.NewReader(tt.body))

			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), tt.actor)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
