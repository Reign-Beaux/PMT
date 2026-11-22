package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/auth/authtest"
	"pmt/internal/features/auth/commands/seedowner"
	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

const ownerPassword = "contrasena-dueno"

// cliente guarda la cookie de sesión entre peticiones, como el navegador.
type cliente struct {
	t       *testing.T
	handler http.Handler
	cookie  *http.Cookie
}

func (c *cliente) do(method, path, body, token string) *httptest.ResponseRecorder {
	c.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == rest.SessionCookie {
			if ck.MaxAge < 0 {
				c.cookie = nil
			} else {
				c.cookie = ck
			}
		}
	}
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("cuerpo %q: %v", rec.Body.String(), err)
	}
	return v
}

func montar(t *testing.T, agents authtest.Agents) (http.Handler, *authtest.Repo) {
	t.Helper()
	repo := authtest.New()
	f := NewWithRepository(repo, agents, Settings{PublicURL: "https://pmt.example.com/", CookieSecure: true})
	if err := f.SeedOwner(context.Background(), seedowner.Command{Email: "Saul@Example.com ", Password: ownerPassword, Username: "saul"}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	f.Register(mux)
	mux.HandleFunc("GET /api/whoami", func(w http.ResponseWriter, r *http.Request) {
		a := rest.ActorFrom(r.Context())
		w.Write([]byte(a.Name + "/" + string(a.Profile)))
	})
	return rest.Authenticate(f, IsPublic)(mux), repo
}

func TestFlujoCompleto(t *testing.T) {
	handler, repo := montar(t, authtest.Agents{"dbp-tl": true})
	dueno := &cliente{t: t, handler: handler}

	if rec := dueno.do("GET", "/api/whoami", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión: %d", rec.Code)
	}
	if rec := dueno.do("POST", "/api/auth/login", `{"email":"saul@example.com","password":"mala-clave"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("contraseña mala: %d", rec.Code)
	}
	rec := dueno.do("POST", "/api/auth/login", `{"email":" SAUL@example.com","password":"`+ownerPassword+`"}`, "")
	if rec.Code != http.StatusOK || dueno.cookie == nil || !dueno.cookie.HttpOnly || !dueno.cookie.Secure {
		t.Fatalf("login: %d %s cookie=%+v", rec.Code, rec.Body, dueno.cookie)
	}
	if got := dueno.do("GET", "/api/whoami", "", "").Body.String(); got != "saul/User" {
		t.Fatalf("actor con cookie = %q", got)
	}
	me := decode[map[string]any](t, dueno.do("GET", "/api/auth/me", "", ""))
	if me["username"] != "saul" || me["is_instance_owner"] != true || me["email"] != "saul@example.com" {
		t.Fatalf("me = %v", me)
	}

	// Token de API: se muestra una vez y sirve como Bearer.
	rec = dueno.do("POST", "/api/tokens", `{"name":" laptop "}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("crear token: %d %s", rec.Code, rec.Body)
	}
	creado := decode[map[string]any](t, rec)
	token, _ := creado["token"].(string)
	if !strings.HasPrefix(token, "pmt_") || creado["name"] != "laptop" {
		t.Fatalf("token = %v", creado)
	}
	if got := dueno.do("GET", "/api/whoami", "", token).Body.String(); got != "saul/User" {
		t.Fatalf("actor con token = %q", got)
	}
	if me["profile"] != "User" {
		t.Fatalf("perfil con cookie = %v", me["profile"])
	}

	// Perfil por sesión: con token, X-PMT-Profile decide el perfil; con la
	// cookie se ignora. Un usuario con perfil sigue siendo él para me, pero las
	// operaciones de cuenta siguen siendo solo de User.
	conPerfil := func(path, value, bearer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set(rest.ProfileHeader, value)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		} else {
			req.AddCookie(dueno.cookie)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if got := conPerfil("/api/whoami", "tl", token).Body.String(); got != "saul/TL" {
		t.Fatalf("token con perfil = %q", got)
	}
	if got := conPerfil("/api/whoami", "tl", "").Body.String(); got != "saul/User" {
		t.Fatalf("cookie con perfil = %q", got)
	}
	if rec := conPerfil("/api/whoami", "jefe", token); rec.Code != http.StatusBadRequest {
		t.Fatalf("perfil desconocido: %d %s", rec.Code, rec.Body)
	}
	meTL := decode[map[string]any](t, conPerfil("/api/auth/me", "TL", token))
	if meTL["username"] != "saul" || meTL["profile"] != "TL" {
		t.Fatalf("me con perfil = %v", meTL)
	}
	if rec := conPerfil("/api/tokens", "tl", token); rec.Code != http.StatusForbidden {
		t.Fatalf("las cuentas siguen siendo de User: %d %s", rec.Code, rec.Body)
	}
	lista := dueno.do("GET", "/api/tokens", "", "").Body.String()
	if strings.Contains(lista, token) || !strings.Contains(lista, `"laptop"`) {
		t.Fatalf("la lista no debe traer el token: %s", lista)
	}

	// Invitación: el enlace usa app.public_url y el invitado elige username.
	rec = dueno.do("POST", "/api/invitations", `{"email":"Ana@Example.com"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("invitar: %d %s", rec.Code, rec.Body)
	}
	inv := decode[map[string]any](t, rec)
	url, _ := inv["url"].(string)
	tok, ok := strings.CutPrefix(url, "https://pmt.example.com/invitacion/")
	if !ok || tok == "" || inv["email"] != "ana@example.com" {
		t.Fatalf("invitación = %v", inv)
	}
	ana := &cliente{t: t, handler: handler}
	if st := decode[map[string]any](t, ana.do("GET", "/api/invitations/"+tok, "", "")); st["valid"] != true || st["email"] != "ana@example.com" {
		t.Fatalf("consulta pública = %v", st)
	}
	if rec := ana.do("POST", "/api/invitations/"+tok+"/accept", `{"name":"Ana","username":"dbp-tl","password":"contrasena-ana"}`, ""); rec.Code != http.StatusConflict {
		t.Fatalf("username de agente: %d %s", rec.Code, rec.Body)
	}
	if rec := ana.do("POST", "/api/invitations/"+tok+"/accept", `{"name":"Ana","username":"Ana","password":"contrasena-ana"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("username con mayúsculas: %d %s", rec.Code, rec.Body)
	}
	rec = ana.do("POST", "/api/invitations/"+tok+"/accept", `{"name":" Ana ","username":"ana","password":"contrasena-ana"}`, "")
	if rec.Code != http.StatusCreated || ana.cookie == nil {
		t.Fatalf("aceptar: %d %s", rec.Code, rec.Body)
	}
	if got := ana.do("GET", "/api/whoami", "", "").Body.String(); got != "ana/User" {
		t.Fatalf("actor de ana = %q", got)
	}
	if rec := ana.do("POST", "/api/invitations/"+tok+"/accept", `{"name":"Otra","username":"otra","password":"contrasena-otra"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("reusar invitación: %d", rec.Code)
	}
	if rec := ana.do("POST", "/api/invitations", `{"email":"x@example.com"}`, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("ana no invita: %d", rec.Code)
	}
	if rec := ana.do("GET", "/api/users", "", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("ana no lista usuarios: %d", rec.Code)
	}
	tokenAna := decode[map[string]any](t, ana.do("POST", "/api/tokens", `{"name":"ci"}`, ""))["token"].(string)

	// Desactivar: ni cookie ni token entran; reactivar devuelve la cookie nueva, no el token.
	usuarios := decode[[]map[string]any](t, dueno.do("GET", "/api/users", "", ""))
	if len(usuarios) != 2 {
		t.Fatalf("usuarios = %v", usuarios)
	}
	var anaID, duenoID string
	for _, u := range usuarios {
		if u["username"] == "ana" {
			anaID = u["id"].(string)
		} else {
			duenoID = u["id"].(string)
		}
	}
	if rec := dueno.do("POST", "/api/users/"+duenoID+"/disable", "", ""); rec.Code != http.StatusConflict {
		t.Fatalf("desactivar al dueño: %d", rec.Code)
	}
	if rec := dueno.do("POST", "/api/users/"+anaID+"/disable", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("desactivar: %d %s", rec.Code, rec.Body)
	}
	if rec := ana.do("GET", "/api/whoami", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("cookie de desactivada: %d", rec.Code)
	}
	if rec := ana.do("GET", "/api/whoami", "", tokenAna); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token de desactivada: %d", rec.Code)
	}
	if rec := ana.do("POST", "/api/auth/login", `{"email":"ana@example.com","password":"contrasena-ana"}`, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("login de desactivada: %d", rec.Code)
	}
	if rec := dueno.do("POST", "/api/users/"+anaID+"/enable", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("reactivar: %d", rec.Code)
	}
	if rec := ana.do("POST", "/api/auth/login", `{"email":"ana@example.com","password":"contrasena-ana"}`, ""); rec.Code != http.StatusOK {
		t.Fatalf("login reactivada: %d", rec.Code)
	}
	if rec := ana.do("GET", "/api/whoami", "", tokenAna); rec.Code != http.StatusUnauthorized {
		t.Fatalf("el token revocado no vuelve: %d", rec.Code)
	}

	// Revocar y cerrar sesión.
	id := creado["id"].(string)
	if rec := dueno.do("DELETE", "/api/tokens/"+id, "", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("revocar: %d", rec.Code)
	}
	if rec := dueno.do("GET", "/api/whoami", "", token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token revocado: %d", rec.Code)
	}
	sesiones := repo.Sessions()
	if rec := dueno.do("POST", "/api/auth/logout", "", ""); rec.Code != http.StatusNoContent || dueno.cookie != nil {
		t.Fatalf("logout: %d", rec.Code)
	}
	if repo.Sessions() != sesiones-1 {
		t.Fatal("logout debe borrar la sesión")
	}
	if rec := dueno.do("GET", "/api/whoami", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("después de logout: %d", rec.Code)
	}
}

func TestCambiarContrasenaCierraLasDemasSesiones(t *testing.T) {
	handler, _ := montar(t, nil)
	a := &cliente{t: t, handler: handler}
	b := &cliente{t: t, handler: handler}
	login := `{"email":"saul@example.com","password":"` + ownerPassword + `"}`
	a.do("POST", "/api/auth/login", login, "")
	b.do("POST", "/api/auth/login", login, "")
	token := decode[map[string]any](t, a.do("POST", "/api/tokens", `{"name":"x"}`, ""))["token"].(string)

	if rec := a.do("POST", "/api/auth/me/password", `{"current_password":"`+ownerPassword+`","new_password":"nueva-clave"}`, token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("con token no: %d", rec.Code)
	}
	if rec := a.do("POST", "/api/auth/me/password", `{"current_password":"otra","new_password":"nueva-clave"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("actual mala: %d", rec.Code)
	}
	if rec := a.do("POST", "/api/auth/me/password", `{"current_password":"`+ownerPassword+`","new_password":"corta"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("nueva corta: %d", rec.Code)
	}
	if rec := a.do("POST", "/api/auth/me/password", `{"current_password":"`+ownerPassword+`","new_password":"nueva-clave"}`, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("cambiar: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do("GET", "/api/whoami", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("la sesión propia sigue: %d", rec.Code)
	}
	if rec := b.do("GET", "/api/whoami", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("la otra sesión se cierra: %d", rec.Code)
	}
	if rec := b.do("POST", "/api/auth/login", `{"email":"saul@example.com","password":"nueva-clave"}`, ""); rec.Code != http.StatusOK {
		t.Fatalf("login con la nueva: %d", rec.Code)
	}
}

func TestInvitacionVencidaYCorreoRegistrado(t *testing.T) {
	handler, repo := montar(t, nil)
	dueno := &cliente{t: t, handler: handler}
	dueno.do("POST", "/api/auth/login", `{"email":"saul@example.com","password":"`+ownerPassword+`"}`, "")

	if rec := dueno.do("POST", "/api/invitations", `{"email":"saul@example.com"}`, ""); rec.Code != http.StatusConflict {
		t.Fatalf("correo registrado: %d", rec.Code)
	}
	if rec := dueno.do("POST", "/api/invitations", `{"email":"sin-arroba"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("correo inválido: %d", rec.Code)
	}
	repo.AddInvitation("vieja@example.com", "token-viejo", time.Now().Add(-time.Hour))
	invitado := &cliente{t: t, handler: handler}
	if st := decode[map[string]any](t, invitado.do("GET", "/api/invitations/token-viejo", "", "")); st["valid"] != false {
		t.Fatalf("vencida = %v", st)
	}
	if rec := invitado.do("POST", "/api/invitations/token-viejo/accept", `{"name":"V","username":"vieja","password":"contrasena-vieja"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("aceptar vencida: %d", rec.Code)
	}
	if rec := invitado.do("GET", "/api/invitations/no-existe", "", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("inexistente: %d", rec.Code)
	}
}

func TestIsPublic(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		public       bool
	}{
		{"POST", "/api/auth/login", true},
		{"POST", "/api/auth/logout", true},
		{"GET", "/api/auth/login", false},
		{"GET", "/api/auth/me", false},
		{"GET", "/api/invitations/abc", true},
		{"POST", "/api/invitations/abc/accept", true},
		{"GET", "/api/invitations", false},
		{"GET", "/api/invitations/", false},
		{"POST", "/api/invitations", false},
		{"DELETE", "/api/invitations/abc", false},
		{"POST", "/api/invitations/abc/otra", false},
		{"GET", "/api/projects", false},
	} {
		if got := IsPublic(httptest.NewRequest(tc.method, tc.path, nil)); got != tc.public {
			t.Errorf("%s %s = %t", tc.method, tc.path, got)
		}
	}
}

// TestTokenConPerfilConservaElUsuario: el perfil declarado no cambia el usuario
// cuya membresía usa el actor, y InstanceOwnerID da al dueño cuando existe.
func TestTokenConPerfilConservaElUsuario(t *testing.T) {
	ctx := context.Background()
	repo := authtest.New()
	f := NewWithRepository(repo, authtest.Agents{}, Settings{PublicURL: "https://pmt.example.com/"})
	if id, err := f.InstanceOwnerID(ctx); err != nil || id != uuid.Nil {
		t.Fatalf("sin dueño: %v %v", id, err)
	}
	dueno := repo.AddUser("saul@example.com", "saul", true, "contrasena")
	ana := repo.AddUser("ana@example.com", "ana", false, "contrasena")
	token, _ := domain.NewAPIToken()
	repo.CreateAPIToken(ctx, uuid.New(), ana.ID, "x", domain.HashToken(token))

	actor, err := f.AuthenticateToken(ctx, token, kernel.ProfileTL)

	if err != nil || actor != (kernel.Actor{Name: "ana", Profile: kernel.ProfileTL, UserID: ana.ID}) {
		t.Fatalf("actor = %+v, err = %v", actor, err)
	}
	if id, err := f.InstanceOwnerID(ctx); err != nil || id != dueno.ID {
		t.Fatalf("dueño = %v, err = %v", id, err)
	}
}
