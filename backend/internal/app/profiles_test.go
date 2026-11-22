package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pmt/internal/features/profiles"
	"pmt/internal/features/profiles/catalog"
	"pmt/internal/kernel"
	"pmt/internal/platform/mcp"
	"pmt/internal/platform/rest"
)

// rutasConPerfiles arma las rutas reales con la feature de perfiles: sus rutas
// REST y get_profile en el MCP.
func rutasConPerfiles() http.Handler {
	api := http.NewServeMux()
	tools := mcp.NewServer()
	profiles.Register(api, tools)
	return routes(api, tools, credentials{users: usuarios{}, agents: agentes{}, owner: duenoDeLaInstancia{}}, pingOK{}, frontendDePrueba())
}

// inicializar manda el initialize del MCP y devuelve las instrucciones que recibe.
func inicializar(t *testing.T, handler http.Handler, target, authorization string) string {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"prueba","version":"1"}}}`
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", authorization)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var res struct {
		Result struct {
			Instructions string `json:"instructions"`
			ServerInfo   struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || res.Result.ServerInfo.Name != "pmt" {
		t.Fatalf("respuesta = %s (%v)", rec.Body, err)
	}
	return res.Result.Instructions
}

func servido(t *testing.T, profile kernel.Profile) string {
	t.Helper()
	text, ok := catalog.Served(profile)
	if !ok {
		t.Fatalf("%s sin texto", profile)
	}
	return text
}

// TestInitializeTraeInstruccionesCortas: con perfil declarado por un usuario,
// las instrucciones son unas líneas que nombran el perfil y piden get_profile y
// .pmt/entorno.md, por debajo de los ~4 000 caracteres que muestra Claude Code.
// Sin perfil, o con token de agente, no hay.
func TestInitializeTraeInstruccionesCortas(t *testing.T) {
	handler := rutasConPerfiles()
	for _, tc := range []struct {
		name, target, authorization string
		profile                     string
	}{
		{"usuario como TL", "/mcp?profile=tl", "Bearer pmt_ana", "TL (Tech Lead)"},
		{"usuario como QA", "/mcp?profile=QA", "Bearer pmt_ana", "QA (Quality Assurance)"},
		{"usuario como PC", "/mcp?profile=pc", "Bearer pmt_ana", "PC (Project Coordinator)"},
		{"usuario como DevBackend", "/mcp?profile=devbackend", "Bearer pmt_ana", "DevBackend (Developer Backend)"},
		{"usuario sin perfil", "/mcp", "Bearer pmt_ana", ""},
		{"usuario como user", "/mcp?profile=user", "Bearer pmt_ana", ""},
		{"agente sin declarar su perfil", "/mcp", "Bearer token-de-agente", ""},
		{"agente declarando su perfil", "/mcp?profile=tl", "Bearer token-de-agente", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := inicializar(t, handler, tc.target, tc.authorization)
			if tc.profile == "" {
				if got != "" {
					t.Errorf("no debe traer instrucciones: %.120q", got)
				}
				return
			}
			if !strings.Contains(got, "perfil "+tc.profile) || !strings.Contains(got, "get_profile") || !strings.Contains(got, catalog.EnvFile) {
				t.Errorf("instrucciones = %q", got)
			}
			if len(got) >= 1000 {
				t.Errorf("instrucciones de %d caracteres: deben ser cortas", len(got))
			}
		})
	}
}

// TestGetProfile: devuelve el texto servido del perfil de la sesión, el mismo de
// REST; sin perfil, un error que dice cómo declararlo; con token de agente, el
// perfil del agente.
func TestGetProfile(t *testing.T) {
	handler := rutasConPerfiles()
	for _, tc := range []struct {
		name, target, authorization string
		isError                     bool
		want                        string
	}{
		{"usuario como TL", "/mcp?profile=tl", "Bearer pmt_ana", false, servido(t, kernel.ProfileTL)},
		{"usuario como PC", "/mcp?profile=pc", "Bearer pmt_ana", false, servido(t, kernel.ProfilePC)},
		{"usuario como QAP", "/mcp?profile=qap", "Bearer pmt_ana", false, servido(t, kernel.ProfileQAP)},
		{"usuario sin perfil", "/mcp", "Bearer pmt_ana", true, "invalid: esta sesión no tiene perfil: declara uno al conectar el MCP con ?profile=<perfil>"},
		{"usuario como user", "/mcp?profile=user", "Bearer pmt_ana", true, "?profile=<perfil>"},
		{"agente sin declarar su perfil", "/mcp", "Bearer token-de-agente", false, servido(t, kernel.ProfileTL)},
		{"agente declarando su perfil", "/mcp?profile=tl", "Bearer token-de-agente", false, servido(t, kernel.ProfileTL)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := llamarHerramienta(handler, tc.target, tc.authorization, "", "get_profile", `{}`)
			var res struct {
				Result struct {
					IsError bool `json:"isError"`
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"result"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || len(res.Result.Content) != 1 {
				t.Fatalf("respuesta = %.300s (%v)", rec.Body, err)
			}
			text := res.Result.Content[0].Text
			if res.Result.IsError != tc.isError {
				t.Fatalf("isError = %v: %.200q", res.Result.IsError, text)
			}
			if tc.isError && !strings.Contains(text, tc.want) || !tc.isError && text != tc.want {
				t.Errorf("texto = %.200q", text)
			}
		})
	}
	if tl := servido(t, kernel.ProfileTL); !strings.HasPrefix(tl, "# Eres TL") {
		t.Errorf("TL abre con %.30q", tl)
	}
}

func TestRutasDePerfiles(t *testing.T) {
	handler := rutasConPerfiles()
	for _, tc := range []struct {
		name, path, authorization, cookie string
		code                              int
		contentType, body                 string
	}{
		{"lista con token", "/api/profiles", "Bearer pmt_ana", "", http.StatusOK, "application/json", `{"name":"devbackend","profile":"DevBackend","description":"Developer Backend"}`},
		{"lista con la cookie", "/api/profiles", "", "sesion-ana", http.StatusOK, "application/json", `"name":"tl"`},
		{"perfil TL", "/api/profiles/tl", "Bearer pmt_ana", "", http.StatusOK, "text/markdown", "# Eres TL (Tech Lead)"},
		{"perfil sin distinguir mayúsculas", "/api/profiles/DevFrontend", "Bearer token-de-agente", "", http.StatusOK, "text/markdown", "# Eres DevFrontend"},
		{"perfil inexistente", "/api/profiles/xx", "Bearer pmt_ana", "", http.StatusNotFound, "application/json", "perfil no encontrado"},
		{"user no tiene perfil", "/api/profiles/user", "Bearer pmt_ana", "", http.StatusNotFound, "application/json", "perfil no encontrado"},
		{"lista sin credenciales", "/api/profiles", "", "", http.StatusUnauthorized, "application/json", "autenticación requerida"},
		{"perfil sin credenciales", "/api/profiles/tl", "", "", http.StatusUnauthorized, "application/json", "autenticación requerida"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := llamarRuta(handler, http.MethodGet, tc.path, tc.authorization, tc.cookie)
			if rec.Code != tc.code || !strings.HasPrefix(rec.Header().Get("Content-Type"), tc.contentType) || !strings.Contains(rec.Body.String(), tc.body) {
				t.Errorf("status = %d, tipo = %q, cuerpo = %.200s", rec.Code, rec.Header().Get("Content-Type"), rec.Body)
			}
		})
	}
	if rec := llamarRuta(handler, http.MethodGet, "/api/profiles/qa", "Bearer pmt_ana", ""); rec.Body.String() != servido(t, kernel.ProfileQA) {
		t.Error("REST y get_profile deben servir el mismo texto")
	}
}

func llamarRuta(handler http.Handler, method, path, authorization, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: rest.SessionCookie, Value: cookie})
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
