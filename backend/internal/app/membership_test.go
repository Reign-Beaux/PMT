package app

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"

	projects "pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/mcp"
	"pmt/internal/platform/rest"
)

// sinProyecto son las herramientas y rutas que no reciben el ID de un proyecto
// ni de algo suyo: crean un proyecto o listan los del actor, que ya filtra.
var (
	herramientasSinProyecto = []string{"create_project", "import_project", "list_projects"}
	rutasSinProyecto        = []string{"GET /api/projects", "POST /api/projects"}
)

// ningunaMembresia: todo elemento existe y el actor no es miembro de su proyecto.
type ningunaMembresia struct{}

func (ningunaMembresia) RoleOf(context.Context, uuid.UUID, string, uuid.UUID) (projects.Role, bool, error) {
	return "", true, nil
}

func (ningunaMembresia) HoldOf(context.Context, string, uuid.UUID) (string, kernel.Hold, error) {
	return "", kernel.Hold{}, nil
}

// soloViewer: todo elemento existe y el actor es viewer de su proyecto.
type soloViewer struct{}

func (soloViewer) RoleOf(context.Context, uuid.UUID, string, uuid.UUID) (projects.Role, bool, error) {
	return projects.RoleViewer, true, nil
}

func (soloViewer) HoldOf(context.Context, string, uuid.UUID) (string, kernel.Hold, error) {
	return "", kernel.Hold{}, nil
}

var (
	usuarioConPerfil = kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: anaID, Channel: kernel.ChannelToken}
	usuarioWeb       = kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: anaID, Channel: kernel.ChannelSession}
)

// rechazos son los dos casos en que ninguna herramienta ni ruta con el ID de un
// proyecto o de algo suyo debe llegar a su handler: quien no es miembro recibe
// not_found, y un viewer con token, forbidden aunque solo lea.
var rechazos = []struct {
	name    string
	members membership
	status  int
	prefix  string
	ok      func(message string) bool
}{
	{"no miembro", ningunaMembresia{}, http.StatusNotFound, "not_found: ", esNoEncontrado},
	{"viewer con token", soloViewer{}, http.StatusForbidden, "forbidden: ", func(m string) bool { return m == projects.ErrViewerOnlyWeb.Error() }},
}

// esNoEncontrado indica si un mensaje es el not_found de algún tipo de elemento,
// el mismo que daría la operación con un ID inexistente.
func esNoEncontrado(message string) bool {
	for _, err := range notFound {
		if message == err.Error() {
			return true
		}
	}
	return false
}

// conectar arma las herramientas reales con la membresía protegida y un pool
// nulo: una herramienta que llegue a la base entra en pánico y la prueba lo
// reporta como membresía no comprobada.
func conectar(t *testing.T, members membership, actor kernel.Actor) (*sdk.ClientSession, http.Handler) {
	t.Helper()
	ctx := context.Background()
	mux := http.NewServeMux()
	server := mcp.NewServer()
	registerFeatures(mux, server, nil)
	api := protect(mux, server, guard{members: members})
	mcp.ActAs(server, actor)
	server.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (result sdk.Result, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("llegó a la base sin comprobar la membresía: %v", r)
				}
			}()
			return next(ctx, method, req)
		}
	})
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session, api
}

func texto(result *sdk.CallToolResult) string {
	if len(result.Content) > 0 {
		if content, ok := result.Content[0].(*sdk.TextContent); ok {
			return content.Text
		}
	}
	return ""
}

// TestTodaHerramientaCompruebaLaMembresia recorre toda herramienta registrada:
// con el ID de un proyecto ajeno, o de algo suyo, debe responder not_found sin
// llegar a su handler; y si el actor es viewer y llega con token, forbidden, lea
// o escriba. Solo se salvan las que no reciben ninguno.
func TestTodaHerramientaCompruebaLaMembresia(t *testing.T) {
	for _, rechazo := range rechazos {
		t.Run(rechazo.name, func(t *testing.T) {
			session, _ := conectar(t, rechazo.members, usuarioConPerfil)
			ctx := context.Background()
			checked := 0
			for tool, err := range session.Tools(ctx, nil) {
				if err != nil {
					t.Fatal(err)
				}
				args := sampleArguments(t, tool)
				if slices.Contains(herramientasSinProyecto, tool.Name) {
					for name := range args {
						if strings.HasSuffix(name, "_id") || strings.HasSuffix(name, "_ids") {
							t.Errorf("%s recibe %s: debe comprobar la membresía", tool.Name, name)
						}
					}
					continue
				}
				checked++
				t.Run(tool.Name, func(t *testing.T) {
					result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: tool.Name, Arguments: args})
					if err != nil {
						t.Fatal(err)
					}
					message, ok := strings.CutPrefix(texto(result), rechazo.prefix)
					if !result.IsError || !ok || !rechazo.ok(message) {
						t.Errorf("no comprobó la membresía: %q", texto(result))
					}
				})
			}
			if checked < 60 {
				t.Fatalf("solo se revisaron %d herramientas", checked)
			}
		})
	}
}

// TestViewerSoloLee: con cualquier herramienta, un viewer que llega con la
// cookie del sitio web pasa la membresía si lee y recibe forbidden si escribe.
func TestViewerSoloLee(t *testing.T) {
	session, _ := conectar(t, soloViewer{}, usuarioWeb)
	g := guard{members: soloViewer{}}
	ctx := context.Background()
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(herramientasSinProyecto, tool.Name) {
			continue
		}
		raw, _ := json.Marshal(sampleArguments(t, tool))
		err := g.tool(ctx, usuarioWeb, tool.Name, raw)
		if readOnly(tool.Name) && err != nil {
			t.Errorf("%s: el viewer debe poder leer: %v", tool.Name, err)
		}
		if !readOnly(tool.Name) && err != projects.ErrViewerCannotWrite {
			t.Errorf("%s: el viewer no debe escribir: %v", tool.Name, err)
		}
	}
}

// rutaRegistrada es una ruta REST de una feature, tomada de su código.
type rutaRegistrada struct {
	pattern string
	file    string
}

// rutasDelCodigo lee las rutas que registra cada rest.go de las features, salvo
// las de auth, que son de cuentas y no de proyectos.
func rutasDelCodigo(t *testing.T) []rutaRegistrada {
	t.Helper()
	var routes []rutaRegistrada
	err := filepath.WalkDir("../features", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(path) != "rest.go" || strings.HasPrefix(path, "../features/auth/") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle") {
				return true
			}
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				pattern, _ := strconv.Unquote(lit.Value)
				routes = append(routes, rutaRegistrada{pattern: pattern, file: path})
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return routes
}

// contrato es la parte de los *.openapi.yaml que la prueba usa para armar cada
// petición: los parámetros de query y las propiedades del cuerpo.
type contrato struct {
	operations map[string]map[string]any // "GET /api/..." → operación
	schemas    map[string]any
}

func leerContratos(t *testing.T) contrato {
	t.Helper()
	paths, err := filepath.Glob("../../api/features/*/*.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c := contrato{operations: map[string]map[string]any{}, schemas: map[string]any{}}
	for _, path := range paths {
		if strings.Contains(path, "/auth/") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		feature := filepath.Base(filepath.Dir(path))
		if components, ok := doc["components"].(map[string]any); ok {
			if schemas, ok := components["schemas"].(map[string]any); ok {
				for name, schema := range schemas {
					c.schemas[feature+"/"+name] = schema
				}
			}
		}
		pathsDoc, _ := doc["paths"].(map[string]any)
		for route, ops := range pathsDoc {
			for method, op := range ops.(map[string]any) {
				if operation, ok := op.(map[string]any); ok {
					operation["x-feature"] = feature
					c.operations[strings.ToUpper(method)+" "+route] = operation
				}
			}
		}
	}
	return c
}

// properties junta las propiedades de un esquema, siguiendo $ref, allOf, oneOf y anyOf.
func (c contrato) properties(feature string, schema any) map[string]any {
	out := map[string]any{}
	s, ok := schema.(map[string]any)
	if !ok {
		return out
	}
	if ref, ok := s["$ref"].(string); ok {
		return c.properties(feature, c.schemas[feature+"/"+strings.TrimPrefix(ref, "#/components/schemas/")])
	}
	if props, ok := s["properties"].(map[string]any); ok {
		for name, prop := range props {
			out[name] = prop
		}
	}
	for _, key := range []string{"allOf", "oneOf", "anyOf"} {
		list, _ := s[key].([]any)
		for _, item := range list {
			for name, prop := range c.properties(feature, item) {
				out[name] = prop
			}
		}
	}
	return out
}

// valorDeID da un valor de muestra para un campo con ID, o false si no es uno.
func valorDeID(name string) (any, bool) {
	switch {
	case name == "item_kind":
		return "task", true
	case strings.HasSuffix(name, "_ids"):
		return []string{uuid.NewString()}, true
	case strings.HasSuffix(name, "_id"):
		return uuid.NewString(), true
	}
	return nil, false
}

// peticion arma la petición de una ruta con los IDs que acepta según su contrato.
func (c contrato) peticion(t *testing.T, pattern string, actor kernel.Actor) *http.Request {
	t.Helper()
	operation, ok := c.operations[pattern]
	if !ok {
		t.Fatalf("%s no está en ningún *.openapi.yaml", pattern)
	}
	feature, _ := operation["x-feature"].(string)
	method, path, _ := strings.Cut(pattern, " ")
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, "{") {
			name := strings.Trim(segment, "{}")
			segments[i] = uuid.NewString()
			if name == "agent" || name == "username" {
				segments[i] = "bea"
			}
		}
	}
	query := []string{}
	params, _ := operation["parameters"].([]any)
	for _, p := range params {
		param, _ := p.(map[string]any)
		name, _ := param["name"].(string)
		if param["in"] != "query" {
			continue
		}
		if value, ok := valorDeID(name); ok {
			query = append(query, name+"="+fmt.Sprint(value))
		}
	}
	target := strings.Join(segments, "/")
	if len(query) > 0 {
		target += "?" + strings.Join(query, "&")
	}
	body := map[string]any{}
	if requestBody, ok := operation["requestBody"].(map[string]any); ok {
		content, _ := requestBody["content"].(map[string]any)
		jsonContent, _ := content["application/json"].(map[string]any)
		for name := range c.properties(feature, jsonContent["schema"]) {
			if value, ok := valorDeID(name); ok {
				body[name] = value
			}
		}
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, target, strings.NewReader(string(raw)))
	return req.WithContext(rest.WithActor(req.Context(), actor))
}

// TestTodaRutaCompruebaLaMembresia recorre las rutas que registran las features
// en su código: con el ID de un proyecto ajeno, o de algo suyo, en la ruta, la
// query o el cuerpo que acepta según su contrato, debe responder 404 sin llegar a
// su handler; y si el actor es viewer y llega con token, 403, lea o escriba.
// También exige que contrato y código tengan las mismas rutas.
func TestTodaRutaCompruebaLaMembresia(t *testing.T) {
	contracts := leerContratos(t)
	routes := rutasDelCodigo(t)
	if len(routes) < 60 {
		t.Fatalf("solo se encontraron %d rutas", len(routes))
	}
	registered := map[string]bool{}
	for _, route := range routes {
		registered[route.pattern] = true
	}
	for pattern := range contracts.operations {
		if !registered[pattern] {
			t.Errorf("%s está en el contrato pero ninguna feature la registra", pattern)
		}
	}
	for _, rechazo := range rechazos {
		t.Run(rechazo.name, func(t *testing.T) {
			_, api := conectar(t, rechazo.members, usuarioConPerfil)
			for _, route := range routes {
				if slices.Contains(rutasSinProyecto, route.pattern) {
					continue
				}
				t.Run(route.pattern, func(t *testing.T) {
					req := contracts.peticion(t, route.pattern, usuarioConPerfil)
					rec := httptest.NewRecorder()
					func() {
						defer func() {
							if r := recover(); r != nil {
								t.Errorf("llegó a la base sin comprobar la membresía (%s): %v", route.file, r)
							}
						}()
						api.ServeHTTP(rec, req)
					}()
					var body struct {
						Error string `json:"error"`
					}
					_ = json.Unmarshal(rec.Body.Bytes(), &body)
					if rec.Code != rechazo.status || !rechazo.ok(body.Error) {
						t.Errorf("no comprobó la membresía: %d %s", rec.Code, rec.Body)
					}
				})
			}
		})
	}
}

// personas y sus roles en dos proyectos: P1 de ana, con carlos de editor y bea de
// viewer; P2 del dueño de la instancia, saul, con cuyo rol actúan los tokens de
// agente. dan no es miembro de nada.
var (
	beaID    = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	carlosID = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")
	danID    = uuid.MustParse("00000000-0000-0000-0000-0000000000d1")
	etapaP1  = uuid.MustParse("00000000-0000-0000-0000-000000000101")
	etapaP2  = uuid.MustParse("00000000-0000-0000-0000-000000000201")
	proyP1   = uuid.MustParse("00000000-0000-0000-0000-000000000100")
	proyP2   = uuid.MustParse("00000000-0000-0000-0000-000000000200")
)

type membresiasDePrueba struct{}

func (membresiasDePrueba) RoleOf(_ context.Context, userID uuid.UUID, _ string, id uuid.UUID) (projects.Role, bool, error) {
	roles := map[uuid.UUID]map[uuid.UUID]projects.Role{
		proyP1: {anaID: projects.RoleOwner, carlosID: projects.RoleEditor, beaID: projects.RoleViewer},
		proyP2: {saulID: projects.RoleOwner},
	}
	project := map[uuid.UUID]uuid.UUID{etapaP1: proyP1, etapaP2: proyP2, proyP1: proyP1, proyP2: proyP2}[id]
	if project == uuid.Nil {
		return "", false, nil
	}
	return roles[project][userID], true, nil
}

func (membresiasDePrueba) HoldOf(context.Context, string, uuid.UUID) (string, kernel.Hold, error) {
	return "", kernel.Hold{}, nil
}

// personasConToken autentica el token pmt_<nombre> de cada persona como usuario,
// y su cookie web_<nombre>.
type personasConToken struct{}

// AuthenticateSession acepta la cookie web_<nombre>: la misma persona con la
// cookie del sitio web.
func (p personasConToken) AuthenticateSession(ctx context.Context, session string) (kernel.Actor, error) {
	name, ok := strings.CutPrefix(session, "web_")
	if !ok {
		return kernel.Actor{}, kernel.ErrSessionInvalid
	}
	actor, err := p.AuthenticateToken(ctx, "pmt_"+name, "")
	if err != nil {
		return kernel.Actor{}, kernel.ErrSessionInvalid
	}
	return actor, nil
}

func (personasConToken) AuthenticateToken(_ context.Context, token string, declared kernel.Profile) (kernel.Actor, error) {
	ids := map[string]uuid.UUID{"pmt_ana": anaID, "pmt_bea": beaID, "pmt_carlos": carlosID, "pmt_dan": danID}
	id, ok := ids[token]
	if !ok {
		return kernel.Actor{}, kernel.ErrUnauthenticated
	}
	actor := kernel.AsUser(strings.TrimPrefix(token, "pmt_"), declared)
	actor.UserID = id
	return actor, nil
}

// TestRolesPorPersona recorre la petición entera, por REST y por MCP, con una
// lectura y una escritura sobre una etapa: owner y editor leen y escriben con
// token o cookie; el viewer con token no llega (forbidden) y con la cookie solo
// lee; quien no es miembro no la ve, y el token de agente actúa con la membresía
// del dueño de la instancia. La cookie solo sirve en REST: el MCP exige token.
func TestRolesPorPersona(t *testing.T) {
	api := http.NewServeMux()
	ok := func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) }
	api.HandleFunc("GET /api/stages/{id}", ok)
	api.HandleFunc("PUT /api/stages/{id}", ok)
	tools := mcp.NewServer()
	type stageInput struct {
		StageID string `json:"stage_id"`
	}
	handler := func(context.Context, *sdk.CallToolRequest, stageInput) (*sdk.CallToolResult, struct{}, error) {
		return nil, struct{}{}, nil
	}
	sdk.AddTool(tools, &sdk.Tool{Name: "list_cosas"}, handler)
	sdk.AddTool(tools, &sdk.Tool{Name: "update_cosa"}, handler)
	protected := protect(api, tools, guard{members: membresiasDePrueba{}})
	all := routes(protected, tools, credentials{users: personasConToken{}, agents: agentes{}, owner: duenoDeLaInstancia{}}, pingOK{}, frontendDePrueba())

	for _, tc := range []struct {
		name, token string
		stage       uuid.UUID
		read, write int
	}{
		{"owner", "pmt_ana", etapaP1, http.StatusOK, http.StatusOK},
		{"editor", "pmt_carlos", etapaP1, http.StatusOK, http.StatusOK},
		{"viewer con token", "pmt_bea", etapaP1, http.StatusForbidden, http.StatusForbidden},
		{"owner con cookie", "web_ana", etapaP1, http.StatusOK, http.StatusOK},
		{"editor con cookie", "web_carlos", etapaP1, http.StatusOK, http.StatusOK},
		{"viewer con cookie", "web_bea", etapaP1, http.StatusOK, http.StatusForbidden},
		{"no miembro con cookie", "web_dan", etapaP1, http.StatusNotFound, http.StatusNotFound},
		{"no miembro", "pmt_dan", etapaP1, http.StatusNotFound, http.StatusNotFound},
		{"token de agente en un proyecto del dueño", "token-de-agente", etapaP2, http.StatusOK, http.StatusOK},
		{"token de agente en un proyecto ajeno al dueño", "token-de-agente", etapaP1, http.StatusNotFound, http.StatusNotFound},
		{"etapa inexistente", "pmt_ana", uuid.New(), http.StatusNotFound, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for method, want := range map[string]int{http.MethodGet: tc.read, http.MethodPut: tc.write} {
				req := httptest.NewRequest(method, "/api/stages/"+tc.stage.String(), strings.NewReader(`{}`))
				cookie := strings.HasPrefix(tc.token, "web_")
				if cookie {
					req.AddCookie(&http.Cookie{Name: rest.SessionCookie, Value: tc.token})
				} else {
					req.Header.Set("Authorization", "Bearer "+tc.token)
				}
				rec := httptest.NewRecorder()
				all.ServeHTTP(rec, req)
				if rec.Code != want {
					t.Errorf("REST %s: status = %d, quiero %d: %s", method, rec.Code, want, rec.Body)
				}
				if want == http.StatusForbidden && !cookie && !strings.Contains(rec.Body.String(), projects.ErrViewerOnlyWeb.Error()) {
					t.Errorf("REST %s: quiero el mensaje del viewer con token: %s", method, rec.Body)
				}
			}
			if strings.HasPrefix(tc.token, "web_") {
				return
			}
			for tool, want := range map[string]int{"list_cosas": tc.read, "update_cosa": tc.write} {
				rec := llamarHerramienta(all, "/mcp", "Bearer "+tc.token, "", tool, `{"stage_id":"`+tc.stage.String()+`"}`)
				body := strings.ReplaceAll(rec.Body.String(), `\"`, `"`)
				prefix := map[int]string{http.StatusOK: `"isError":`, http.StatusForbidden: "forbidden: " + projects.ErrViewerOnlyWeb.Error(), http.StatusNotFound: "not_found: etapa no encontrada"}[want]
				failed := strings.Contains(body, `"isError":true`)
				if want == http.StatusOK && failed || want != http.StatusOK && !strings.Contains(body, prefix) {
					t.Errorf("MCP %s: quiero %d: %s", tool, want, body)
				}
			}
		})
	}
}

// TestEventosSoloDeProyectosQueVe: la visibilidad que usa /api/events sigue la
// membresía, el viewer solo ve los eventos con la cookie y el token de agente ve
// lo del dueño de la instancia.
func TestEventosSoloDeProyectosQueVe(t *testing.T) {
	g := guard{members: membresiasDePrueba{}}
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		actor   kernel.Actor
		project uuid.UUID
		want    bool
	}{
		{"owner", kernel.Actor{UserID: anaID}, proyP1, true},
		{"editor", kernel.Actor{UserID: carlosID}, proyP1, true},
		{"viewer con token", kernel.Actor{UserID: beaID, Channel: kernel.ChannelToken}, proyP1, false},
		{"viewer sin canal cuenta como token", kernel.Actor{UserID: beaID}, proyP1, false},
		{"viewer con cookie", kernel.Actor{UserID: beaID, Channel: kernel.ChannelSession}, proyP1, true},
		{"editor con cookie", kernel.Actor{UserID: carlosID, Channel: kernel.ChannelSession}, proyP1, true},
		{"no miembro", kernel.Actor{UserID: danID}, proyP1, false},
		{"token de agente, proyecto del dueño", agenteC, proyP2, true},
		{"token de agente, proyecto ajeno", agenteC, proyP1, false},
		{"sin usuario", agente, proyP2, false},
		{"proyecto inexistente", kernel.Actor{UserID: anaID}, uuid.New(), false},
	} {
		if got, err := g.visible(ctx, tc.actor, tc.project); err != nil || got != tc.want {
			t.Errorf("%s: ve = %v, err = %v", tc.name, got, err)
		}
	}
}

// TestIDsDeCadaCampo: los campos con ID de herramientas y rutas se resuelven al
// tipo correcto, y un valor que no es UUID lo deja para la operación.
func TestIDsDeCadaCampo(t *testing.T) {
	id := uuid.New()
	for _, tc := range []struct {
		name     string
		fields   map[string]any
		planKind string
		want     []target
	}{
		{"comentario de un bug", map[string]any{"item_kind": "bug", "item_id": id.String()}, "", []target{{kindBug, id}}},
		{"item_kind desconocido", map[string]any{"item_kind": "epic", "item_id": id.String()}, "", nil},
		{"plan de pruebas", map[string]any{"plan_id": id.String()}, kindTestPlan, []target{{kindTestPlan, id}}},
		{"plan de desarrollo", map[string]any{"plan_id": id.String()}, kindDevPlan, []target{{kindDevPlan, id}}},
		{"lote de tareas", map[string]any{"task_ids": []any{id.String(), "malo"}}, "", []target{{kindTask, id}}},
		{"ID inválido", map[string]any{"stage_id": "malo"}, "", nil},
		{"en orden", map[string]any{"task_id": id.String(), "project_id": id.String()}, "", []target{{kindProject, id}, {kindTask, id}}},
	} {
		if got := targetsOf(tc.fields, tc.planKind); !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v", tc.name, got)
		}
	}
}
