package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	bugs "pmt/internal/features/bugs/domain"
	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// carlos es editor y escribe con token; bea, otra editora, tiene tomada la etapa
// o el bug de cada elemento que recibe la operación.
var (
	carlosConToken = kernel.Actor{Name: "carlos", Profile: kernel.ProfileUser, UserID: carlosID, UserName: "carlos", Channel: kernel.ChannelToken}
	tomadoPorBea   = kernel.Hold{HolderID: beaID, TakenBy: "bea"}
)

// contenidos dice qué tipos de elemento quedan dentro de una etapa o de un bug.
// El plan de desarrollo y el hallazgo pueden ser de los dos.
var contenidos = map[string][]string{
	kindStage: {kindStage, kindTask, kindTestPlan, kindDevPlan, kindFinding},
	kindBug:   {kindBug, kindDevPlan, kindFinding},
}

// tomado: el actor es editor de todo proyecto y la etapa (o el bug) que contiene a
// cada elemento la tiene holder.
type tomado struct {
	kind   string
	holder kernel.Hold
}

func (tomado) RoleOf(context.Context, uuid.UUID, string, uuid.UUID) (projects.Role, bool, error) {
	return projects.RoleEditor, true, nil
}

func (t tomado) HoldOf(_ context.Context, kind string, _ uuid.UUID) (string, kernel.Hold, error) {
	if slices.Contains(contenidos[t.kind], kind) {
		return t.kind, t.holder, nil
	}
	return "", kernel.Hold{}, nil
}

// mensajeDe es el conflict de cada bloqueo tomado por bea.
var mensajeDe = map[string]string{
	kindStage: stages.ErrTakenBy("bea").Error(),
	kindBug:   bugs.ErrTakenBy("bea").Error(),
}

// escriturasDelProyecto son las herramientas y rutas de escritura que solo
// reciben el proyecto: no tocan ninguna etapa ni bug existentes, así que el
// bloqueo no las detiene. Agregar una etapa (add_stage, import_stage) se permite
// aunque otra persona trabaje otra.
var (
	herramientasDelProyecto = []string{
		"add_project_member", "add_stage", "change_product_version", "change_project_status",
		"create_bug", "create_stage", "delete_project", "import_stage", "release_project",
		"remove_project_member", "update_project", "update_project_member",
	}
	rutasDelProyecto = []string{
		"DELETE /api/projects/{id}", "DELETE /api/projects/{id}/members/{username}",
		"POST /api/projects/{id}/members", "POST /api/projects/{id}/product-version",
		"POST /api/projects/{id}/release", "POST /api/projects/{project_id}/bugs",
		"POST /api/projects/{project_id}/stages", "POST /api/projects/{project_id}/stages/add",
		"PUT /api/projects/{id}", "PUT /api/projects/{id}/members/{username}",
		"PUT /api/projects/{id}/status",
	}
)

// argumentosPara son los argumentos de muestra de una herramienta: con item_kind,
// del tipo que contiene el bloqueo, para que comentarios y hallazgos lo prueben
// sobre una etapa (tarea) y sobre un bug.
func argumentosPara(t *testing.T, tool *sdk.Tool, kind string) map[string]any {
	args := sampleArguments(t, tool)
	if _, ok := args["item_kind"]; ok && kind == kindBug {
		args["item_kind"] = "bug"
	}
	return args
}

// recibe indica si unos elementos incluyen alguno dentro de una etapa o un bug
// del tipo dado, y si solo incluyen proyectos.
func recibe(targets []target, kind string) (contained, onlyProject bool) {
	onlyProject = true
	for _, t := range targets {
		if t.kind != kindProject {
			onlyProject = false
		}
		if slices.Contains(contenidos[kind], t.kind) {
			contained = true
		}
	}
	return contained, onlyProject
}

// TestTodaEscrituraRespetaElBloqueo recorre toda herramienta de escritura: sobre
// una etapa tomada por otra persona, o sobre algo suyo, y sobre un bug tomado por
// otra persona, debe responder conflict con el nombre de quien la tiene, sin
// llegar a su handler. Solo se salvan release_stage y release_bug, y las que solo
// reciben el proyecto, que deben ser exactamente herramientasDelProyecto.
func TestTodaEscrituraRespetaElBloqueo(t *testing.T) {
	for _, kind := range []string{kindStage, kindBug} {
		t.Run(kind, func(t *testing.T) {
			members := tomado{kind: kind, holder: tomadoPorBea}
			session, _ := conectar(t, members, carlosConToken)
			g := guard{members: members}
			ctx := context.Background()
			checked := 0
			var projectOnly []string
			for tool, err := range session.Tools(ctx, nil) {
				if err != nil {
					t.Fatal(err)
				}
				if readOnly(tool.Name) || slices.Contains(herramientasSinProyecto, tool.Name) {
					continue
				}
				args := argumentosPara(t, tool, kind)
				contained, onlyProject := recibe(targetsOf(args, toolPlanKind(tool.Name)), kind)
				if onlyProject {
					projectOnly = append(projectOnly, tool.Name)
				}
				if slices.Contains(releases, tool.Name) {
					raw, _ := json.Marshal(args)
					if err := g.tool(ctx, carlosConToken, tool.Name, raw); err != nil {
						t.Errorf("%s: liberar no puede rechazarse por estar tomada: %v", tool.Name, err)
					}
					continue
				}
				if !contained {
					continue
				}
				checked++
				t.Run(tool.Name, func(t *testing.T) {
					result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: tool.Name, Arguments: args})
					if err != nil {
						t.Fatal(err)
					}
					if !result.IsError || texto(result) != "conflict: "+mensajeDe[kind] {
						t.Errorf("no respetó el bloqueo: %q", texto(result))
					}
				})
			}
			if kind == kindStage {
				slices.Sort(projectOnly)
				if !slices.Equal(projectOnly, herramientasDelProyecto) {
					t.Errorf("escrituras que solo reciben el proyecto: %v; decide si respetan el bloqueo y actualiza herramientasDelProyecto", projectOnly)
				}
			}
			minimum := map[string]int{kindStage: 29, kindBug: 7}[kind]
			if checked < minimum {
				t.Fatalf("solo se revisaron %d herramientas", checked)
			}
		})
	}
}

// TestTodaRutaDeEscrituraRespetaElBloqueo hace lo mismo con las rutas REST que
// registran las features.
func TestTodaRutaDeEscrituraRespetaElBloqueo(t *testing.T) {
	contracts := leerContratos(t)
	routes := rutasDelCodigo(t)
	for _, kind := range []string{kindStage, kindBug} {
		t.Run(kind, func(t *testing.T) {
			members := tomado{kind: kind, holder: tomadoPorBea}
			_, api := conectar(t, members, carlosConToken)
			g := guard{members: members}
			checked := 0
			var projectOnly []string
			for _, route := range routes {
				method, _, _ := strings.Cut(route.pattern, " ")
				if method == http.MethodGet || slices.Contains(rutasSinProyecto, route.pattern) {
					continue
				}
				req := contracts.peticion(t, route.pattern, carlosConToken)
				if kind == kindBug {
					req = conItemKindBug(t, req)
				}
				setPattern(req, route.pattern)
				_, targets := routeTargets(req, route.pattern)
				contained, onlyProject := recibe(targets, kind)
				if onlyProject {
					projectOnly = append(projectOnly, route.pattern)
				}
				if slices.Contains(releases, route.pattern) {
					if err := g.route(req, route.pattern); err != nil {
						t.Errorf("%s: liberar no puede rechazarse por estar tomada: %v", route.pattern, err)
					}
					continue
				}
				if !contained {
					continue
				}
				checked++
				t.Run(route.pattern, func(t *testing.T) {
					req := contracts.peticion(t, route.pattern, carlosConToken)
					if kind == kindBug {
						req = conItemKindBug(t, req)
					}
					rec := httptest.NewRecorder()
					func() {
						defer func() {
							if r := recover(); r != nil {
								t.Errorf("llegó a la base sin respetar el bloqueo (%s): %v", route.file, r)
							}
						}()
						api.ServeHTTP(rec, req)
					}()
					var body struct {
						Error string `json:"error"`
					}
					_ = json.Unmarshal(rec.Body.Bytes(), &body)
					if rec.Code != http.StatusConflict || body.Error != mensajeDe[kind] {
						t.Errorf("no respetó el bloqueo: %d %s", rec.Code, rec.Body)
					}
				})
			}
			if kind == kindStage {
				slices.Sort(projectOnly)
				if !slices.Equal(projectOnly, rutasDelProyecto) {
					t.Errorf("rutas de escritura que solo reciben el proyecto: %v; decide si respetan el bloqueo y actualiza rutasDelProyecto", projectOnly)
				}
			}
			minimum := map[string]int{kindStage: 29, kindBug: 6}[kind]
			if checked < minimum {
				t.Fatalf("solo se revisaron %d rutas", checked)
			}
		})
	}
}

// conItemKindBug cambia item_kind del cuerpo a bug, si lo tiene.
func conItemKindBug(t *testing.T, req *http.Request) *http.Request {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["item_kind"]; ok {
		body["item_kind"] = "bug"
	}
	raw, _ := json.Marshal(body)
	clone := httptest.NewRequest(req.Method, req.URL.String(), strings.NewReader(string(raw)))
	return clone.WithContext(req.Context())
}

// setPattern llena los valores de la ruta de una petición armada a mano, como lo
// haría el mux, para leer sus IDs sin servirla.
func setPattern(req *http.Request, pattern string) {
	_, path, _ := strings.Cut(pattern, " ")
	patternSegments := strings.Split(path, "/")
	segments := strings.Split(req.URL.Path, "/")
	for i, segment := range patternSegments {
		if strings.HasPrefix(segment, "{") && i < len(segments) {
			req.SetPathValue(strings.Trim(segment, "{}"), segments[i])
		}
	}
}

// TestBloqueoPorPersona: la persona que tiene tomada la etapa escribe con
// cualquier perfil, también con un token de agente que actúa como ella; una
// etapa libre la escribe cualquiera; y leer nunca se bloquea.
func TestBloqueoPorPersona(t *testing.T) {
	ctx := context.Background()
	stage := `{"stage_id":"` + uuid.NewString() + `","reason":"x"}`
	tests := []struct {
		name   string
		holder kernel.Hold
		actor  kernel.Actor
		tool   string
		want   error
	}{
		{"la misma persona como TL", tomadoPorBea, kernel.Actor{Name: "bea", Profile: kernel.ProfileTL, UserID: beaID}, "change_stage_status", nil},
		{"un agente de la persona", tomadoPorBea, kernel.Actor{Name: "dbp-tl", Profile: kernel.ProfileTL, UserID: beaID, UserName: "bea"}, "change_stage_status", nil},
		{"otra persona", tomadoPorBea, carlosConToken, "change_stage_status", stages.ErrTakenBy("bea")},
		{"otra persona con el owner", tomadoPorBea, kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: anaID}, "update_stage", stages.ErrTakenBy("bea")},
		{"etapa libre", kernel.Hold{}, carlosConToken, "change_stage_status", nil},
		{"otra persona lee", tomadoPorBea, carlosConToken, "list_stage_status_changes", nil},
		{"el owner libera", tomadoPorBea, kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: anaID}, "release_stage", nil},
	}
	for _, tt := range tests {
		g := guard{members: tomado{kind: kindStage, holder: tt.holder}}
		err := g.tool(ctx, tt.actor, tt.tool, json.RawMessage(stage))
		if (err == nil) != (tt.want == nil) || err != nil && err.Error() != tt.want.Error() {
			t.Errorf("%s: err = %v, quiero %v", tt.name, err, tt.want)
		}
	}
}

// TestBloqueoNoCambiaLaMembresia: la membresía se revisa antes que el bloqueo, así
// que quien no es miembro sigue recibiendo not_found y el viewer, forbidden.
func TestBloqueoNoCambiaLaMembresia(t *testing.T) {
	ctx := context.Background()
	args := json.RawMessage(`{"stage_id":"` + uuid.NewString() + `"}`)
	if err := (guard{members: ningunaMembresia{}}).tool(ctx, carlosConToken, "update_stage", args); err != stages.ErrStageNotFound {
		t.Errorf("no miembro: %v", err)
	}
	if err := (guard{members: soloViewer{}}).tool(ctx, carlosConToken, "update_stage", args); err != projects.ErrViewerOnlyWeb {
		t.Errorf("viewer: %v", err)
	}
}
