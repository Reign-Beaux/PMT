package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
	"pmt/internal/features/workflow/commands/startstage"
	workflow "pmt/internal/features/workflow/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/mcp"
	"pmt/internal/platform/rest"
)

func TestCredentialsResuelvenElPerfilDeclarado(t *testing.T) {
	c := credentials{users: usuarios{}, agents: agentes{}}
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		token    string
		declared kernel.Profile
		want     kernel.Actor
		kind     kernel.ErrorKind
	}{
		{"usuario sin perfil", "pmt_ana", "", ana, 0},
		{"usuario como TL", "pmt_ana", kernel.ProfileTL, kernel.Actor{Name: "ana", Profile: kernel.ProfileTL}, 0},
		{"usuario como PC", "pmt_ana", kernel.ProfilePC, kernel.Actor{Name: "ana", Profile: kernel.ProfilePC}, 0},
		{"agente sin perfil", "token-de-agente", "", agente, 0},
		{"agente con su perfil", "token-de-agente", kernel.ProfileTL, agente, 0},
		{"agente con otro perfil", "token-de-agente", kernel.ProfilePP, kernel.Actor{}, kernel.KindInvalid},
		{"token inválido", "pmt_otro", kernel.ProfileTL, kernel.Actor{}, kernel.KindUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := c.AuthenticateToken(ctx, tc.token, tc.declared)
			var business *kernel.Error
			if tc.kind != 0 && (!errors.As(err, &business) || business.Kind != tc.kind) {
				t.Fatalf("err = %v, quiero tipo %d", err, tc.kind)
			}
			if tc.kind == 0 && err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("actor = %+v", got)
			}
		})
	}
	if got, _ := c.AuthenticateSession(ctx, "sesion-ana"); got != ana {
		t.Errorf("la cookie siempre es User: %+v", got)
	}
}

func TestRoutesAPIConPerfilDeclarado(t *testing.T) {
	handler := rutasConActor()
	for _, tc := range []struct {
		name, path, authorization, cookie, profile string
		code                                       int
		body                                       string
	}{
		{"token de usuario como TL", "/api/projects", "Bearer pmt_ana", "", "tl", http.StatusOK, "ana/TL"},
		{"token de usuario como QA en eventos", "/api/events", "Bearer pmt_ana", "", "QA", http.StatusOK, "ana/QA"},
		{"user equivale a no declarar", "/api/projects", "Bearer pmt_ana", "", "user", http.StatusOK, "ana/User"},
		{"perfil desconocido", "/api/projects", "Bearer pmt_ana", "", "xx", http.StatusBadRequest, "perfil desconocido: usa pp, pc, tl, dp, devfrontend, devbackend, cr, qap, qa o user"},
		{"la cookie ignora la cabecera", "/api/projects", "", "sesion-ana", "tl", http.StatusOK, "ana/User"},
		{"agente con su perfil", "/api/projects", "Bearer token-de-agente", "", "TL", http.StatusOK, "dbp-tl/TL"},
		{"agente con otro perfil", "/api/projects", "Bearer token-de-agente", "", "pp", http.StatusBadRequest, "declara ese perfil o ninguno"},
		{"sin credenciales con perfil", "/api/projects", "", "", "tl", http.StatusUnauthorized, "autenticación requerida"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: rest.SessionCookie, Value: tc.cookie})
			}
			req.Header.Set(rest.ProfileHeader, tc.profile)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
				t.Errorf("status = %d, cuerpo = %q", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRoutesMCPConPerfilDeclarado(t *testing.T) {
	handler := rutasConActor()
	for _, tc := range []struct {
		name, target, authorization string
		code                        int
		body                        string
	}{
		{"token de usuario como PP", "/mcp?profile=pp", "Bearer pmt_ana", http.StatusOK, `"name":"ana","profile":"PP"`},
		{"token de usuario como DevFrontend", "/mcp?profile=DEVFRONTEND", "Bearer pmt_ana", http.StatusOK, `"name":"ana","profile":"DevFrontend"`},
		{"perfil desconocido", "/mcp?profile=xx", "Bearer pmt_ana", http.StatusBadRequest, "perfil desconocido"},
		{"agente con su perfil", "/mcp?profile=tl", "Bearer token-de-agente", http.StatusOK, `"name":"dbp-tl","profile":"TL"`},
		{"agente con otro perfil", "/mcp?profile=qa", "Bearer token-de-agente", http.StatusBadRequest, "declara ese perfil o ninguno"},
		{"sin token con perfil", "/mcp?profile=xx", "", http.StatusUnauthorized, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := llamarHerramienta(handler, tc.target, tc.authorization, "", "whoami", `{}`)
			if rec.Code != tc.code || !strings.Contains(strings.ReplaceAll(rec.Body.String(), `\"`, `"`), tc.body) {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

// etapaDeAna es una etapa en stand_by cuyo TL es la usuaria ana.
type etapaDeAna struct {
	stage   stages.Stage
	project projects.Project
	started bool
}

func (r *etapaDeAna) Project(context.Context, uuid.UUID) (projects.Project, error) {
	return r.project, nil
}

func (r *etapaDeAna) Stage(context.Context, uuid.UUID) (stages.Stage, error) {
	return r.stage, nil
}

func (r *etapaDeAna) Tasks(context.Context, uuid.UUID) ([]tasks.Task, error) {
	return []tasks.Task{{ID: uuid.New(), Status: tasks.StatusBacklog}}, nil
}

func (r *etapaDeAna) Start(context.Context, workflow.StartPlan) error {
	r.started = true
	return nil
}

// TestUsuarioComoTLIniciaSuEtapa recorre la petición entera: el token de usuario
// con ?profile=tl llega a start_stage como ana/TL y, siendo la TL de la etapa,
// la inicia. Con ?profile=pc, o sin perfil, la misma usuaria no puede.
func TestUsuarioComoTLIniciaSuEtapa(t *testing.T) {
	for _, tc := range []struct {
		target  string
		started bool
		prefix  string
	}{
		{"/mcp?profile=tl", true, ""},
		{"/mcp?profile=pc", false, "forbidden: "},
		{"/mcp", false, "forbidden: "},
	} {
		t.Run(tc.target, func(t *testing.T) {
			repo := &etapaDeAna{
				stage:   stages.Stage{ID: uuid.New(), Name: "Pagos", Status: stages.StatusStandBy, Lead: "ana"},
				project: projects.Project{ID: uuid.New(), Status: projects.StatusStandBy},
			}
			tools := mcp.NewServer()
			startstage.New(repo).RegisterTool(tools)
			handler := routes(http.NewServeMux(), tools, credentials{users: usuarios{}, agents: agentes{}}, pingOK{}, frontendDePrueba())

			rec := llamarHerramienta(handler, tc.target, "Bearer pmt_ana", "", "start_stage", `{"stage_id":"`+repo.stage.ID.String()+`"}`)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body)
			}
			if repo.started != tc.started {
				t.Fatalf("iniciada = %t: %s", repo.started, rec.Body)
			}
			body := rec.Body.String()
			if tc.started && (!strings.Contains(body, `in_dev`) || strings.Contains(body, `"isError":true`)) {
				t.Errorf("cuerpo = %s", body)
			}
			if !tc.started && !strings.Contains(body, tc.prefix) {
				t.Errorf("se esperaba %q: %s", tc.prefix, body)
			}
		})
	}
}
