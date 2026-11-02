// Package file lee los ficheros YAML de la feature imports. Lo comparten las
// slices, que no se importan entre sí.
package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	tasks "pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

const (
	// DescriptionRequired es el mensaje de una descripción vacía.
	DescriptionRequired = "la descripción es obligatoria"

	commentInText      = "el texto contiene un comentario; ponlo entre comillas"
	invalidYAMLMessage = "el YAML no es válido"
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

// Project es la entrada de un proyecto completo.
type Project struct {
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Stages      []Stage `yaml:"stages"`
}

// Stage es la entrada de una etapa con sus tareas.
type Stage struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Tasks       []Task `yaml:"tasks"`
}

// Task es la entrada de una tarea.
type Task struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type projectFile struct {
	Project Project `yaml:"project"`
}

type stageFile struct {
	Stage Stage `yaml:"stage"`
}

// ReadProject lee y revisa un fichero con raíz project.
func ReadProject(path string) (Project, error) {
	raw, err := read(path)
	if err != nil {
		return Project{}, err
	}
	var parsed projectFile
	if err := decode(raw, &parsed); err != nil {
		return Project{}, err
	}
	if err := checkComments(raw, projectComments); err != nil {
		return Project{}, err
	}
	return parsed.Project, nil
}

// ReadStage lee y revisa un fichero con raíz stage.
func ReadStage(path string) (Stage, error) {
	raw, err := read(path)
	if err != nil {
		return Stage{}, err
	}
	var parsed stageFile
	if err := decode(raw, &parsed); err != nil {
		return Stage{}, err
	}
	if err := checkComments(raw, stageComments); err != nil {
		return Stage{}, err
	}
	return parsed.Stage, nil
}

// BuildTasks valida las tareas de la etapa. prefix antecede a su ubicación en los errores.
func (s Stage) BuildTasks(stageID uuid.UUID, prefix string) ([]tasks.Task, error) {
	built := make([]tasks.Task, 0, len(s.Tasks))
	for i, entry := range s.Tasks {
		where := fmt.Sprintf("%starea %d", prefix, i+1)
		task, err := tasks.NewTask(stageID, entry.Title, entry.Description)
		if err != nil {
			return nil, Located(where, err.Error())
		}
		if Blank(entry.Description) {
			return nil, Located(where, DescriptionRequired)
		}
		built = append(built, task)
	}
	return built, nil
}

// Blank indica si un texto está vacío.
func Blank(text string) bool {
	return strings.TrimSpace(text) == ""
}

// Located crea un error con su ubicación en el fichero.
func Located(where, message string) error {
	return kernel.Invalid(where + ": " + message)
}

func read(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, kernel.Invalid(fmt.Sprintf("no se pudo leer el fichero: %v", err))
	}
	return raw, nil
}

func decode(raw []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)

	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return kernel.Invalid("el fichero está vacío")
		}
		return invalidYAML(err)
	}

	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return kernel.Invalid("el fichero debe contener un solo documento YAML")
	}
	return nil
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
func checkComments(raw []byte, check func(root *yaml.Node) error) error {
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return invalidYAML(err)
	}
	if len(document.Content) == 0 {
		return nil
	}
	return check(document.Content[0])
}

func projectComments(root *yaml.Node) error {
	project := fieldValue(root, "project")
	if commented(project, "name", "description") {
		return Located("proyecto", commentInText)
	}
	for i, stage := range items(fieldValue(project, "stages")) {
		if err := stageNodeComments(stage, fmt.Sprintf("etapa %d", i+1), fmt.Sprintf("etapa %d, ", i+1)); err != nil {
			return err
		}
	}
	return nil
}

func stageComments(root *yaml.Node) error {
	return stageNodeComments(fieldValue(root, "stage"), "etapa", "")
}

func stageNodeComments(stage *yaml.Node, where, taskPrefix string) error {
	if commented(stage, "name", "description") {
		return Located(where, commentInText)
	}
	for i, task := range items(fieldValue(stage, "tasks")) {
		if commented(task, "title", "description") {
			return Located(fmt.Sprintf("%starea %d", taskPrefix, i+1), commentInText)
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
