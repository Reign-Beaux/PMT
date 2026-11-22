package app

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"

	bugs "pmt/internal/features/bugs/domain"
	devplans "pmt/internal/features/devplans/domain"
	projects "pmt/internal/features/projects/domain"
	qa "pmt/internal/features/qa/domain"
	reviews "pmt/internal/features/reviews/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

// La membresía se comprueba en un solo lugar, antes de cada herramienta MCP y de
// cada ruta REST: toda operación que recibe el ID de un proyecto o de algo suyo
// lo resuelve a su proyecto y exige que quien actúa sea miembro. Así ninguna
// operación nueva puede olvidarla; membership_test.go recorre todas.
//
// - Sin membresía, o con un ID que no existe, responde not_found con el mismo
//   mensaje que daría la operación: el proyecto no existe para esa persona.
// - Un viewer con token (de usuario, por MCP o REST) recibe forbidden en todo:
//   solo consulta desde el sitio web, con la cookie.
// - Un viewer con la cookie que escribe recibe forbidden. Escribe toda
//   herramienta que no empiece con list_ ni get_, y toda ruta que no sea GET.
// - Una escritura sobre una etapa tomada por otra persona, o sobre algo suyo
//   (tareas, planes, hallazgos, comentarios), o sobre un bug tomado por otra
//   persona, responde conflict con el username de quien la tiene. Solo se salvan
//   release_stage y release_bug, con las que el owner la libera.
// - Después, cada operación aplica sus reglas de perfil como siempre.

// Tipos de elemento que resuelve projects.Membership.
const (
	kindProject  = "project"
	kindStage    = "stage"
	kindTask     = "task"
	kindBug      = "bug"
	kindFinding  = "finding"
	kindTestPlan = "test_plan"
	kindDevPlan  = "dev_plan"
)

// idFields son los campos con IDs de un proyecto o de algo suyo, en el orden en
// que se comprueban, con el tipo que nombran. plan_id e item_id dependen de la
// operación y de item_kind.
var idFields = []struct{ name, kind string }{
	{"project_id", kindProject},
	{"stage_id", kindStage},
	{"task_id", kindTask},
	{"task_ids", kindTask},
	{"bug_id", kindBug},
	{"item_id", ""},
	{"finding_id", kindFinding},
	{"plan_id", ""},
}

// itemKinds son los valores de item_kind (comentarios y hallazgos).
var itemKinds = map[string]string{"project": kindProject, "stage": kindStage, "task": kindTask, "bug": kindBug}

// resourceIDs dice qué campo es el {id} de una ruta REST según su primer segmento
// bajo /api/. Las demás rutas con {id} (tokens, usuarios) no son de un proyecto.
var resourceIDs = map[string]string{
	"projects":   "project_id",
	"stages":     "stage_id",
	"tasks":      "task_id",
	"bugs":       "bug_id",
	"findings":   "finding_id",
	"test-plans": "plan_id",
	"dev-plans":  "plan_id",
}

// notFound es el error de cada tipo cuando no existe: el mismo de su feature.
var notFound = map[string]error{
	kindProject:  projects.ErrProjectNotFound,
	kindStage:    stages.ErrStageNotFound,
	kindTask:     tasks.ErrTaskNotFound,
	kindBug:      bugs.ErrBugNotFound,
	kindFinding:  reviews.ErrFindingNotFound,
	kindTestPlan: qa.ErrPlanNotFound,
	kindDevPlan:  devplans.ErrPlanNotFound,
}

// readOnly indica si la herramienta solo lee. Una herramienta nueva que solo lee
// debe seguir esta convención de nombre; cualquier otra cuenta como escritura.
func readOnly(name string) bool {
	return strings.HasPrefix(name, "list_") || strings.HasPrefix(name, "get_")
}

// releases son las herramientas y rutas con las que el owner libera una etapa o
// un bug tomados: no pueden rechazarse por estar tomados.
var releases = []string{"release_stage", "release_bug", "POST /api/stages/{id}/release", "POST /api/bugs/{id}/release"}

// takenBy es el conflict de cada tipo de bloqueo, con el username de quien lo tiene.
var takenBy = map[string]func(username string) error{
	kindStage: stages.ErrTakenBy,
	kindBug:   bugs.ErrTakenBy,
}

// membership da el rol de un usuario en el proyecto de un elemento y quién tiene
// tomada la etapa o el bug que lo contiene. La cumple projects.Membership.
type membership interface {
	RoleOf(ctx context.Context, userID uuid.UUID, kind string, id uuid.UUID) (projects.Role, bool, error)
	HoldOf(ctx context.Context, kind string, id uuid.UUID) (holderKind string, hold kernel.Hold, err error)
}

// access es lo que hace una operación: si escribe y si respeta los bloqueos.
type access struct {
	write     bool
	checkHold bool
}

func accessOf(write bool, name string) access {
	return access{write: write, checkHold: write && !slices.Contains(releases, name)}
}

// guard comprueba la membresía de las herramientas, las rutas y los eventos.
type guard struct {
	members membership
}

type target struct {
	kind string
	id   uuid.UUID
}

// tool revisa una llamada MCP. Unos argumentos que no son un objeto pasan: la
// herramienta los rechaza.
func (g guard) tool(ctx context.Context, actor kernel.Actor, name string, arguments json.RawMessage) error {
	var fields map[string]any
	if len(arguments) > 0 && json.Unmarshal(arguments, &fields) != nil {
		return nil
	}
	return g.verify(ctx, actor, accessOf(!readOnly(name), name), targetsOf(fields, toolPlanKind(name)))
}

// toolPlanKind dice de qué tipo es el plan_id de una herramienta.
func toolPlanKind(name string) string {
	switch {
	case strings.Contains(name, "test_plan"):
		return kindTestPlan
	case strings.Contains(name, "dev_plan"):
		return kindDevPlan
	}
	return ""
}

// route revisa una petición REST: los IDs de su ruta, de su query y de su cuerpo.
func (g guard) route(r *http.Request, pattern string) error {
	write, targets := routeTargets(r, pattern)
	return g.verify(r.Context(), rest.ActorFrom(r.Context()), accessOf(write, pattern), targets)
}

// routeTargets dice si una petición REST escribe y qué elementos recibe.
func routeTargets(r *http.Request, pattern string) (bool, []target) {
	_, path, _ := strings.Cut(pattern, " ")
	segments := strings.Split(strings.Trim(path, "/"), "/")
	resource := ""
	if len(segments) > 1 {
		resource = segments[1]
	}
	planKind := ""
	switch resource {
	case "test-plans":
		planKind = kindTestPlan
	case "dev-plans":
		planKind = kindDevPlan
	}

	fromPath := map[string]any{}
	for _, segment := range segments {
		param, ok := strings.CutPrefix(segment, "{")
		if !ok {
			continue
		}
		param = strings.TrimSuffix(param, "}")
		field := param
		if param == "id" {
			field = resourceIDs[resource]
		}
		if field != "" {
			fromPath[field] = r.PathValue(param)
		}
	}
	fromQuery := map[string]any{}
	for name, values := range r.URL.Query() {
		list := make([]any, 0, len(values))
		for _, v := range values {
			list = append(list, v)
		}
		fromQuery[name] = list
	}

	write := r.Method != http.MethodGet && r.Method != http.MethodHead
	targets := targetsOf(fromPath, planKind)
	targets = append(targets, targetsOf(fromQuery, planKind)...)
	if write {
		targets = append(targets, targetsOf(rest.PeekJSON(r), planKind)...)
	}
	return write, targets
}

// visible indica si el actor ve un proyecto: lo usan los eventos. Un viewer con
// token no lo ve.
func (g guard) visible(ctx context.Context, actor kernel.Actor, projectID uuid.UUID) (bool, error) {
	if actor.UserID == uuid.Nil {
		return false, nil
	}
	role, found, err := g.members.RoleOf(ctx, actor.UserID, kindProject, projectID)
	return found && role.Reaches(actor), err
}

// verify exige que el actor sea miembro del proyecto de cada elemento, que su rol
// le deje llegar con su credencial y que no sea viewer si la operación escribe.
// Después, si escribe, que ninguna etapa ni bug de los elementos lo tenga tomado
// otra persona.
func (g guard) verify(ctx context.Context, actor kernel.Actor, op access, targets []target) error {
	for _, t := range targets {
		if actor.UserID == uuid.Nil {
			return notFound[t.kind]
		}
		role, found, err := g.members.RoleOf(ctx, actor.UserID, t.kind, t.id)
		if err != nil {
			return err
		}
		if !found || role == "" {
			return notFound[t.kind]
		}
		if !role.Reaches(actor) {
			return projects.ErrViewerOnlyWeb
		}
		if op.write && !role.CanWrite() {
			return projects.ErrViewerCannotWrite
		}
	}
	if !op.checkHold {
		return nil
	}
	for _, t := range targets {
		if t.kind == kindProject {
			continue
		}
		holderKind, hold, err := g.members.HoldOf(ctx, t.kind, t.id)
		if err != nil {
			return err
		}
		if conflict, ok := takenBy[holderKind]; ok && hold.HeldByOther(actor) {
			return conflict(hold.TakenBy)
		}
	}
	return nil
}

// targetsOf saca los IDs de unos campos, en el orden de idFields. Un valor que no
// es un UUID se ignora: la operación lo rechaza como inválido.
func targetsOf(fields map[string]any, planKind string) []target {
	var targets []target
	for _, field := range idFields {
		value, ok := fields[field.name]
		if !ok {
			continue
		}
		kind := field.kind
		switch field.name {
		case "plan_id":
			kind = planKind
		case "item_id":
			kind = itemKinds[firstString(fields["item_kind"])]
		}
		if kind == "" {
			continue
		}
		for _, raw := range textsOf(value) {
			if id, err := uuid.Parse(raw); err == nil {
				targets = append(targets, target{kind: kind, id: id})
			}
		}
	}
	return targets
}

// textsOf da los textos de un valor JSON: uno solo o una lista.
func textsOf(value any) []string {
	switch v := value.(type) {
	case string:
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func firstString(value any) string {
	if values := textsOf(value); len(values) > 0 {
		return values[0]
	}
	return ""
}
