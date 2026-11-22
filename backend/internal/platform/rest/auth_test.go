package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pmt/internal/kernel"
)

var (
	agentActor = kernel.Actor{Name: "agente", Profile: kernel.ProfileTL}
	userActor  = kernel.Actor{Name: "ana", Profile: kernel.ProfileUser}
)

type fakeAuthenticator struct{}

func (fakeAuthenticator) AuthenticateSession(_ context.Context, session string) (kernel.Actor, error) {
	switch session {
	case "sesion":
		return userActor, nil
	case "rota":
		return kernel.Actor{}, errors.New("base de datos caída")
	}
	return kernel.Actor{}, kernel.ErrSessionInvalid
}

// AuthenticateToken aplica el perfil declarado como lo hace app: el usuario
// toma cualquiera y el agente solo el suyo.
func (fakeAuthenticator) AuthenticateToken(_ context.Context, token string, declared kernel.Profile) (kernel.Actor, error) {
	switch token {
	case "bueno":
		return kernel.AsAgent(agentActor, declared)
	case "pmt_usuario":
		return kernel.AsUser(userActor.Name, declared), nil
	case "roto":
		return kernel.Actor{}, errors.New("base de datos caída")
	}
	return kernel.Actor{}, kernel.ErrUnauthenticated
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		session string
		profile string
		status  int
		actor   kernel.Actor
	}{
		{"token de usuario con perfil", "Bearer pmt_usuario", "", "tl", http.StatusOK, kernel.Actor{Name: "ana", Profile: kernel.ProfileTL}},
		{"perfil sin distinguir mayúsculas", "Bearer pmt_usuario", "", "DevBackend", http.StatusOK, kernel.Actor{Name: "ana", Profile: kernel.ProfileDevBackend}},
		{"user equivale a no declarar", "Bearer pmt_usuario", "", "USER", http.StatusOK, userActor},
		{"perfil desconocido", "Bearer pmt_usuario", "", "xx", http.StatusBadRequest, kernel.Actor{}},
		{"cookie ignora el perfil", "", "sesion", "tl", http.StatusOK, userActor},
		{"cookie ignora un perfil desconocido", "", "sesion", "xx", http.StatusOK, userActor},
		{"agente con su perfil", "Bearer bueno", "", "tl", http.StatusOK, agentActor},
		{"agente con otro perfil", "Bearer bueno", "", "pp", http.StatusBadRequest, kernel.Actor{}},
		{"sin credenciales con perfil", "", "", "xx", http.StatusUnauthorized, kernel.Actor{}},
		{"sin credenciales", "", "", "", http.StatusUnauthorized, kernel.Actor{}},
		{"variable vacía", "Bearer", "", "", http.StatusUnauthorized, kernel.Actor{}},
		{"vacío con espacio", "Bearer ", "", "", http.StatusUnauthorized, kernel.Actor{}},
		{"token de agente", "Bearer bueno", "", "", http.StatusOK, agentActor},
		{"token de usuario", "Bearer pmt_usuario", "", "", http.StatusOK, userActor},
		{"token inválido", "Bearer malo", "", "", http.StatusUnauthorized, kernel.Actor{}},
		{"otro esquema", "Basic bueno", "", "", http.StatusUnauthorized, kernel.Actor{}},
		{"error técnico", "Bearer roto", "", "", http.StatusInternalServerError, kernel.Actor{}},
		{"cookie", "", "sesion", "", http.StatusOK, userActor},
		{"cookie inválida", "", "vencida", "", http.StatusUnauthorized, kernel.Actor{}},
		{"cookie con error técnico", "", "rota", "", http.StatusInternalServerError, kernel.Actor{}},
		{"la cabecera manda sobre la cookie", "Bearer malo", "sesion", "", http.StatusUnauthorized, kernel.Actor{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got kernel.Actor
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = ActorFrom(r.Context())
			})
			req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			if tt.session != "" {
				req.AddCookie(&http.Cookie{Name: SessionCookie, Value: tt.session})
			}
			if tt.profile != "" {
				req.Header.Set(ProfileHeader, tt.profile)
			}

			rec := httptest.NewRecorder()
			Authenticate(fakeAuthenticator{}, nil)(next).ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("status = %d", rec.Code)
			}
			if got != tt.actor {
				t.Errorf("actor = %+v", got)
			}
		})
	}
}

func TestAuthenticateUnknownProfileListsValid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.Header.Set("Authorization", "Bearer pmt_usuario")
	req.Header.Set(ProfileHeader, "jefe")
	rec := httptest.NewRecorder()

	Authenticate(fakeAuthenticator{}, nil)(http.NotFoundHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "pp, pc, tl, dp, devfrontend, devbackend, cr, qap, qa o user") {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestAuthenticatePublicRoutesSkipCredentials(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		if ActorFrom(r.Context()) != (kernel.Actor{}) {
			t.Error("una ruta pública no debe tener actor")
		}
	})
	public := func(r *http.Request) bool { return r.URL.Path == "/api/auth/login" }
	rec := httptest.NewRecorder()

	Authenticate(fakeAuthenticator{}, public)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", nil))

	if !called || rec.Code != http.StatusOK {
		t.Errorf("status = %d, llamado = %t", rec.Code, called)
	}
}

func TestActorFromWithoutActorIsEmpty(t *testing.T) {
	if got := ActorFrom(context.Background()); got != (kernel.Actor{}) {
		t.Errorf("actor = %+v", got)
	}
}

func TestSetSessionCookie(t *testing.T) {
	rec := httptest.NewRecorder()

	SetSessionCookie(rec, "valor", 60, true)

	cookie := rec.Result().Cookies()[0]
	if cookie.Name != SessionCookie || cookie.Value != "valor" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Errorf("cookie = %+v", cookie)
	}
}

func TestWriteErrorForbidden(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, kernel.Forbidden("no"))

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestWriteErrorUnauthorized(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, kernel.Unauthorized("no"))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d", rec.Code)
	}
}
