package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pmt/internal/features/auth/authtest"
	"pmt/internal/features/auth/domain"
	"pmt/internal/platform/rest"
)

// pedirCodigo pide un código como el CLI, sin credenciales.
func pedirCodigo(t *testing.T, c *cliente, machine string) map[string]any {
	t.Helper()
	rec := c.do("POST", "/api/cli/login", `{"machine":"`+machine+`"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("pedir código: %d %s", rec.Code, rec.Body)
	}
	return decode[map[string]any](t, rec)
}

func sondear(t *testing.T, c *cliente, secret string) map[string]any {
	t.Helper()
	rec := c.do("POST", "/api/cli/login/poll", `{"poll_secret":"`+secret+`"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("sondear: %d %s", rec.Code, rec.Body)
	}
	return decode[map[string]any](t, rec)
}

func entrar(t *testing.T, handler http.Handler) *cliente {
	t.Helper()
	c := &cliente{t: t, handler: handler}
	if rec := c.do("POST", "/api/auth/login", `{"email":"saul@example.com","password":"`+ownerPassword+`"}`, ""); rec.Code != http.StatusOK {
		t.Fatalf("login: %d", rec.Code)
	}
	return c
}

func TestLoginDelCLIAutorizado(t *testing.T) {
	handler, repo := montar(t, authtest.Agents{})
	cli := &cliente{t: t, handler: handler}

	codigo := pedirCodigo(t, cli, " laptop-de-saul ")
	code, _ := codigo["code"].(string)
	secret, _ := codigo["poll_secret"].(string)
	if _, err := domain.NormalizeCLICode(code); err != nil || code != strings.ToUpper(code) {
		t.Fatalf("código = %q", code)
	}
	if codigo["url"] != "https://pmt.example.com/cli?codigo="+code || codigo["interval"] != float64(2) || len(secret) != 64 {
		t.Fatalf("respuesta = %v", codigo)
	}
	guardados := repo.CLILogins()
	if len(guardados) != 1 || guardados[0].Machine != "laptop-de-saul" {
		t.Fatalf("guardado = %+v", guardados)
	}

	if got := sondear(t, cli, secret); got["status"] != "pending" || got["token"] != nil {
		t.Fatalf("pendiente = %v", got)
	}
	if rec := cli.do("GET", "/api/cli/login/"+code, "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("ver sin sesión: %d", rec.Code)
	}
	if rec := cli.do("POST", "/api/cli/login/"+code+"/authorize", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("autorizar sin sesión: %d", rec.Code)
	}

	dueno := entrar(t, handler)
	visto := decode[map[string]any](t, dueno.do("GET", "/api/cli/login/"+strings.ToLower(strings.ReplaceAll(code, "-", "")), "", ""))
	if visto["code"] != code || visto["machine"] != "laptop-de-saul" || visto["token_name"] != "pmt laptop-de-saul" || visto["status"] != "pending" {
		t.Fatalf("visto = %v", visto)
	}
	rec := dueno.do("POST", "/api/cli/login/"+code+"/authorize", "", "")
	if rec.Code != http.StatusOK || decode[map[string]any](t, rec)["status"] != "authorized" {
		t.Fatalf("autorizar: %d %s", rec.Code, rec.Body)
	}
	if rec := dueno.do("POST", "/api/cli/login/"+code+"/authorize", "", ""); rec.Code != http.StatusConflict {
		t.Fatalf("autorizar dos veces: %d", rec.Code)
	}
	if rec := dueno.do("POST", "/api/cli/login/"+code+"/cancel", "", ""); rec.Code != http.StatusConflict {
		t.Fatalf("cancelar autorizado: %d", rec.Code)
	}

	rec = cli.do("POST", "/api/cli/login/poll", `{"poll_secret":"`+secret+`"}`, "")
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("el sondeo no debe guardarse en caché")
	}
	entregado := decode[map[string]any](t, rec)
	token, _ := entregado["token"].(string)
	if entregado["status"] != "authorized" || !strings.HasPrefix(token, "pmt_") || entregado["username"] != "saul" || entregado["token_id"] == nil {
		t.Fatalf("entregado = %v", entregado)
	}
	if got := cli.do("GET", "/api/whoami", "", token).Body.String(); got != "saul/User" {
		t.Fatalf("actor con el token del CLI = %q", got)
	}
	lista := dueno.do("GET", "/api/tokens", "", "").Body.String()
	if !strings.Contains(lista, `"pmt laptop-de-saul"`) || !strings.Contains(lista, entregado["token_id"].(string)) {
		t.Fatalf("el token debe salir en la cuenta: %s", lista)
	}

	// Uso único: el token no se vuelve a entregar.
	if got := sondear(t, cli, secret); got["status"] != "expired" || got["token"] != nil {
		t.Fatalf("segundo sondeo = %v", got)
	}

	// logout del CLI: revoca su propio token con él mismo.
	if rec := cli.do("DELETE", "/api/tokens/"+entregado["token_id"].(string), "", token); rec.Code != http.StatusNoContent {
		t.Fatalf("revocar con el mismo token: %d %s", rec.Code, rec.Body)
	}
	if rec := cli.do("GET", "/api/whoami", "", token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token revocado: %d", rec.Code)
	}
}

func TestLoginDelCLICanceladoYCaducado(t *testing.T) {
	handler, repo := montar(t, authtest.Agents{})
	cli := &cliente{t: t, handler: handler}
	dueno := entrar(t, handler)

	cancelado := pedirCodigo(t, cli, "pc")
	code := cancelado["code"].(string)
	if rec := dueno.do("POST", "/api/cli/login/"+code+"/cancel", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancelar: %d %s", rec.Code, rec.Body)
	}
	if got := sondear(t, cli, cancelado["poll_secret"].(string)); got["status"] != "cancelled" || got["token"] != nil {
		t.Fatalf("cancelado = %v", got)
	}
	if rec := dueno.do("POST", "/api/cli/login/"+code+"/authorize", "", ""); rec.Code != http.StatusConflict {
		t.Fatalf("autorizar cancelado: %d", rec.Code)
	}

	vencido := pedirCodigo(t, cli, "pc")
	repo.ExpireCLILogins()
	if got := sondear(t, cli, vencido["poll_secret"].(string)); got["status"] != "expired" {
		t.Fatalf("vencido = %v", got)
	}
	if got := decode[map[string]any](t, dueno.do("GET", "/api/cli/login/"+vencido["code"].(string), "", "")); got["status"] != "expired" {
		t.Fatalf("visto vencido = %v", got)
	}
	if rec := dueno.do("POST", "/api/cli/login/"+vencido["code"].(string)+"/authorize", "", ""); rec.Code != http.StatusConflict {
		t.Fatalf("autorizar vencido: %d", rec.Code)
	}

	// Autorizado pero sin sondear a tiempo: tampoco entrega el token.
	tarde := pedirCodigo(t, cli, "pc")
	if rec := dueno.do("POST", "/api/cli/login/"+tarde["code"].(string)+"/authorize", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("autorizar: %d", rec.Code)
	}
	repo.ExpireCLILogins()
	if got := sondear(t, cli, tarde["poll_secret"].(string)); got["status"] != "expired" || got["token"] != nil {
		t.Fatalf("autorizado y vencido = %v", got)
	}
}

func TestLoginDelCLIRechazos(t *testing.T) {
	handler, _ := montar(t, authtest.Agents{})
	cli := &cliente{t: t, handler: handler}
	dueno := entrar(t, handler)

	for _, body := range []string{`{}`, `{"machine":"   "}`, `{"machine":"` + strings.Repeat("x", 61) + `"}`, `{"machine":"a\u0007b"}`} {
		if rec := cli.do("POST", "/api/cli/login", body, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("máquina %s: %d", body, rec.Code)
		}
	}
	if rec := cli.do("POST", "/api/cli/login/poll", `{"poll_secret":""}`, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("sondeo sin secreto: %d", rec.Code)
	}
	if rec := cli.do("POST", "/api/cli/login/poll", `{"poll_secret":"inventado"}`, ""); rec.Code != http.StatusNotFound {
		t.Errorf("sondeo con secreto desconocido: %d", rec.Code)
	}
	for _, code := range []string{"ZZZZ-0000", "no-es-codigo"} {
		if rec := dueno.do("GET", "/api/cli/login/"+code, "", ""); rec.Code != http.StatusNotFound {
			t.Errorf("ver %s: %d", code, rec.Code)
		}
		if rec := dueno.do("POST", "/api/cli/login/"+code+"/authorize", "", ""); rec.Code != http.StatusNotFound {
			t.Errorf("autorizar %s: %d", code, rec.Code)
		}
	}

	// Con perfil declarado no se autoriza: es una operación de cuenta.
	codigo := pedirCodigo(t, cli, "pc")
	token := decode[map[string]any](t, dueno.do("POST", "/api/tokens", `{"name":"x"}`, ""))["token"].(string)
	req := httptest.NewRequest("POST", "/api/cli/login/"+codigo["code"].(string)+"/authorize", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(rest.ProfileHeader, "pp")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("autorizar con perfil: %d %s", rec.Code, rec.Body)
	}
}

func TestLoginDelCLILimiteDePendientes(t *testing.T) {
	handler, repo := montar(t, authtest.Agents{})
	cli := &cliente{t: t, handler: handler}
	for i := range domain.MaxPendingCLILogins {
		pedirCodigo(t, cli, fmt.Sprintf("pc-%d", i))
	}
	rec := cli.do("POST", "/api/cli/login", `{"machine":"otra"}`, "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "demasiados") {
		t.Fatalf("pasado el tope: %d %s", rec.Code, rec.Body)
	}
	// Al vencer, se borran y se puede pedir otro.
	repo.ExpireCLILogins()
	pedirCodigo(t, cli, "otra")
	if n := len(repo.CLILogins()); n != 1 {
		t.Fatalf("quedaron %d códigos", n)
	}
}

func TestLoginDelCLIUsuarioDesactivadoNoRecibeToken(t *testing.T) {
	handler, repo := montar(t, authtest.Agents{})
	ana := repo.AddUser("ana@example.com", "ana", false, "contrasena-ana")
	cli := &cliente{t: t, handler: handler}
	anaWeb := &cliente{t: t, handler: handler}
	if rec := anaWeb.do("POST", "/api/auth/login", `{"email":"ana@example.com","password":"contrasena-ana"}`, ""); rec.Code != http.StatusOK {
		t.Fatalf("login ana: %d", rec.Code)
	}
	codigo := pedirCodigo(t, cli, "pc")
	if rec := anaWeb.do("POST", "/api/cli/login/"+codigo["code"].(string)+"/authorize", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("autorizar: %d", rec.Code)
	}
	dueno := entrar(t, handler)
	if rec := dueno.do("POST", "/api/users/"+ana.ID.String()+"/disable", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("desactivar: %d %s", rec.Code, rec.Body)
	}
	if got := sondear(t, cli, codigo["poll_secret"].(string)); got["status"] != "cancelled" || got["token"] != nil {
		t.Fatalf("desactivado = %v", got)
	}
}
