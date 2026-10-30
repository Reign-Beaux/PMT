// Package importproject crea un proyecto completo desde un fichero YAML.
package importproject

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"pmt/internal/features/imports/domain"
	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

const (
	descriptionRequired = "la descripción es obligatoria"
	commentInText       = "el texto contiene un comentario; ponlo entre comillas"
	invalidYAMLMessage  = "el YAML no es válido"
)

// syntaxMessages traduce los problemas de yaml.v3 que tienen una causa concreta.
var syntaxMessages = map[string]string{
	"mapping values are not allowed in this context":        `el texto contiene ":"; ponlo entre comillas`,
	"did not find expected alphabetic or numeric character": "el texto empieza con un carácter especial; ponlo entre comillas",
	"did not find expected key":                             "revisa la indentación o pon el texto entre comillas",
	"found unknown escape character":                        "barra invertida no válida; usa comillas simples",
	"found character that cannot start any token":           "el fichero tiene un tabulador o carácter no válido; usa espacios",
	"found a tab character that violates indentation":       "el fichero tiene un tabulador o carácter no válido; usa espacios",
}

var (
	yamlProblem  = regexp.MustCompile(`^(?:yaml: )?(?:line (\d+): )?(.*)$`)
	unknownField = regexp.MustCompile(`^field (.+) not found in type `)
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Save(ctx context.Context, plan domain.Plan) error
}

// Command contiene los datos de entrada.
type Command struct {
	Path string
}

// Result resume lo creado.
type Result struct {
	ProjectID   uuid.UUID
	ProjectName string
	StageCount  int
	TaskCount   int
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

type file struct {
	Project projectEntry `yaml:"project"`
}

type projectEntry struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Stages      []stageEntry `yaml:"stages"`
}

type stageEntry struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Tasks       []taskEntry `yaml:"tasks"`
}

type taskEntry struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

// Handle valida el fichero completo y después guarda todo como una unidad.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	raw, err := os.ReadFile(cmd.Path)
	if err != nil {
		return Result{}, kernel.Invalid(fmt.Sprintf("no se pudo leer el fichero: %v", err))
	}

	entry, err := decode(raw)
	if err != nil {
		return Result{}, err
	}

	plan, err := buildPlan(entry)
	if err != nil {
		return Result{}, err
	}

	if err := h.repo.Save(ctx, plan); err != nil {
		return Result{}, err
	}

	result := Result{
		ProjectID:   plan.Project.ID,
		ProjectName: plan.Project.Name,
		StageCount:  len(plan.Stages),
	}
	for _, stage := range plan.Stages {
		result.TaskCount += len(stage.Tasks)
	}
	return result, nil
}

func decode(raw []byte) (projectEntry, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)

	var parsed file
	if err := decoder.Decode(&parsed); err != nil {
		if errors.Is(err, io.EOF) {
			return projectEntry{}, kernel.Invalid("el fichero está vacío")
		}
		return projectEntry{}, invalidYAML(err)
	}

	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return projectEntry{}, kernel.Invalid("el fichero debe contener un solo documento YAML")
	}

	if err := checkComments(raw); err != nil {
		return projectEntry{}, err
	}
	return parsed.Project, nil
}

// invalidYAML convierte un error de yaml.v3 en un mensaje para el usuario.
func invalidYAML(err error) error {
	var typeErr *yaml.TypeError
	if !errors.As(err, &typeErr) {
		return kernel.Invalid(translate(err.Error()))
	}

	messages := make([]string, 0, len(typeErr.Errors))
	for _, text := range typeErr.Errors {
		messages = append(messages, translate(text))
	}
	return kernel.Invalid(strings.Join(messages, "; "))
}

func translate(text string) string {
	parts := yamlProblem.FindStringSubmatch(text)
	if parts == nil {
		return invalidYAMLMessage
	}
	line, problem := parts[1], parts[2]

	message, known := syntaxMessages[problem]
	if field := unknownField.FindStringSubmatch(problem); field != nil {
		message, known = fmt.Sprintf("clave desconocida %q", field[1]), true
	}
	if !known {
		message = invalidYAMLMessage
	}
	if line == "" {
		return message
	}
	return "línea " + line + ": " + message
}

// checkComments rechaza los comentarios junto a un texto, porque yaml.v3 los
// descarta sin avisar. Usa yaml.Node porque Decode no los expone en structs.
func checkComments(raw []byte) error {
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return invalidYAML(err)
	}
	if len(document.Content) == 0 {
		return nil
	}

	project := fieldValue(document.Content[0], "project")
	if commented(project, "name", "description") {
		return located("proyecto", commentInText)
	}
	for i, stage := range items(fieldValue(project, "stages")) {
		if commented(stage, "name", "description") {
			return located(fmt.Sprintf("etapa %d", i+1), commentInText)
		}
		for j, task := range items(fieldValue(stage, "tasks")) {
			if commented(task, "title", "description") {
				return located(fmt.Sprintf("etapa %d, tarea %d", i+1, j+1), commentInText)
			}
		}
	}
	return nil
}

// commented indica si alguna de las claves tiene un comentario en su valor.
// En `title: #1 prioridad` no hay valor y yaml.v3 deja el comentario en la clave.
func commented(mapping *yaml.Node, keys ...string) bool {
	for _, key := range keys {
		name, value := field(mapping, key)
		if value == nil {
			continue
		}
		if value.LineComment != "" || (value.Tag == "!!null" && name.LineComment != "") {
			return true
		}
	}
	return false
}

func fieldValue(mapping *yaml.Node, key string) *yaml.Node {
	_, value := field(mapping, key)
	return value
}

func field(mapping *yaml.Node, key string) (name, value *yaml.Node) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i], resolve(mapping.Content[i+1])
		}
	}
	return nil, nil
}

func items(sequence *yaml.Node) []*yaml.Node {
	if sequence == nil || sequence.Kind != yaml.SequenceNode {
		return nil
	}
	resolved := make([]*yaml.Node, 0, len(sequence.Content))
	for _, item := range sequence.Content {
		resolved = append(resolved, resolve(item))
	}
	return resolved
}

func resolve(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.AliasNode {
		return node.Alias
	}
	return node
}

func buildPlan(entry projectEntry) (domain.Plan, error) {
	project, err := projects.NewProject(entry.Name, entry.Description)
	if err != nil {
		return domain.Plan{}, located("proyecto", err.Error())
	}
	if blank(entry.Description) {
		return domain.Plan{}, located("proyecto", descriptionRequired)
	}

	plan := domain.Plan{Project: project}
	for i, stageEntry := range entry.Stages {
		where := fmt.Sprintf("etapa %d", i+1)
		stage, err := stages.NewStage(project.ID, stageEntry.Name, stageEntry.Description)
		if err != nil {
			return domain.Plan{}, located(where, err.Error())
		}
		if blank(stageEntry.Description) {
			return domain.Plan{}, located(where, descriptionRequired)
		}

		stagePlan := domain.StagePlan{Stage: stage}
		for j, taskEntry := range stageEntry.Tasks {
			where := fmt.Sprintf("etapa %d, tarea %d", i+1, j+1)
			task, err := tasks.NewTask(stage.ID, taskEntry.Title, taskEntry.Description)
			if err != nil {
				return domain.Plan{}, located(where, err.Error())
			}
			if blank(taskEntry.Description) {
				return domain.Plan{}, located(where, descriptionRequired)
			}
			stagePlan.Tasks = append(stagePlan.Tasks, task)
		}
		plan.Stages = append(plan.Stages, stagePlan)
	}
	return plan, nil
}

func blank(text string) bool {
	return strings.TrimSpace(text) == ""
}

func located(where, message string) error {
	return kernel.Invalid(where + ": " + message)
}
