package importproject

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pmt/internal/features/imports/domain"
	"pmt/internal/kernel"
)

const validFile = `
project:
  name: Tienda
  description: Venta por internet.
  stages:
    - name: Base de datos
      description: Modelo inicial.
      tasks:
        - title: Tabla de productos
          description: |
            Nombre, precio y existencias.
        - title: Tabla de pedidos
          description: Se relaciona con productos.
    - name: API
      description: Rutas del backend.
      tasks: []
`

type fakeRepo struct {
	saved []domain.Plan
	err   error
}

func (r *fakeRepo) Save(_ context.Context, plan domain.Plan) error {
	r.saved = append(r.saved, plan)
	return r.err
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proyecto.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHandleBuildsPlanInFileOrder(t *testing.T) {
	repo := &fakeRepo{}

	result, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, Path: writeFile(t, validFile)})

	if err != nil {
		t.Fatal(err)
	}
	if result.ProjectName != "Tienda" || result.StageCount != 2 || result.TaskCount != 2 {
		t.Errorf("result = %+v", result)
	}
	plan := repo.saved[0]
	if plan.Project.ID != result.ProjectID || plan.Project.Status != "planning" || plan.Project.ProductVersion != "" {
		t.Errorf("proyecto = %+v", plan.Project)
	}
	first := plan.Stages[0]
	if first.Stage.Name != "Base de datos" || first.Stage.ProjectID != plan.Project.ID || first.Stage.Status != "planning" {
		t.Errorf("etapa = %+v", first.Stage)
	}
	if plan.Stages[1].Stage.Name != "API" || len(plan.Stages[1].Tasks) != 0 {
		t.Errorf("segunda etapa = %+v", plan.Stages[1])
	}
	if first.Tasks[0].Title != "Tabla de productos" || first.Tasks[1].Title != "Tabla de pedidos" {
		t.Errorf("tareas = %+v", first.Tasks)
	}
	if first.Tasks[0].StageID != first.Stage.ID || first.Tasks[0].Status != "backlog" {
		t.Errorf("tarea = %+v", first.Tasks[0])
	}
	if first.Tasks[0].Description != "Nombre, precio y existencias." {
		t.Errorf("descripción = %q", first.Tasks[0].Description)
	}
}

func TestHandleRejectsInvalidFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"vacío", "", "vacío"},
		{"YAML mal formado", "project: [", "línea 1: el YAML no es válido"},
		{"clave desconocida", "project:\n  name: A\n  description: B\n  nombre: C\n", `línea 4: clave desconocida "nombre"`},
		{"clave desconocida en tarea", `
project:
  name: A
  description: B
  stages:
    - name: C
      description: D
      tasks:
        - title: E
          description: F
          status: done
`, `línea 11: clave desconocida "status"`},
		{"varios documentos", "project:\n  name: A\n  description: B\n---\nproject:\n  name: C\n", "un solo documento"},
		{"sin proyecto", "otro: 1\n", `línea 1: clave desconocida "otro"`},
		{"proyecto sin nombre", "project:\n  name: ' '\n  description: B\n", "proyecto: el nombre del proyecto es obligatorio"},
		{"proyecto sin descripción", "project:\n  name: A\n", "proyecto: la descripción es obligatoria"},
		{"etapa sin nombre", `
project:
  name: A
  description: B
  stages:
    - name: C
      description: D
    - description: E
`, "etapa 2: el nombre de la etapa es obligatorio"},
		{"etapa sin descripción", `
project:
  name: A
  description: B
  stages:
    - name: C
`, "etapa 1: la descripción es obligatoria"},
		{"tarea sin título", `
project:
  name: A
  description: B
  stages:
    - name: C
      description: D
      tasks:
        - title: E
          description: F
        - title: ''
          description: G
`, "etapa 1, tarea 2: el título de la tarea es obligatorio"},
		{"tarea sin descripción", `
project:
  name: A
  description: B
  stages:
    - name: C
      description: D
      tasks:
        - title: E
`, "etapa 1, tarea 1: la descripción es obligatoria"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			_, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, Path: writeFile(t, tt.content)})

			assertInvalid(t, err, tt.want)
			if len(repo.saved) != 0 {
				t.Error("no debió guardar nada")
			}
		})
	}
}

func taskFile(title string) string {
	return `project:
  name: A
  description: B
  stages:
    - name: C
      description: D
      tasks:
        - title: E
          description: F
        - title: ` + title + `
          description: G
`
}

func TestHandleTranslatesSyntaxErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"dos puntos", taskFile("Login: pantalla"), `línea 10: el texto contiene ":"; ponlo entre comillas`},
		{"empieza con asterisco", taskFile("*urgente*"), "línea 10: el texto empieza con un carácter especial; ponlo entre comillas"},
		{"empieza con corchete", taskFile("[API] rutas"), "línea 9: revisa la indentación o pon el texto entre comillas"},
		{"indentación", "project:\n  name: A\n description: B\n", "línea 2: revisa la indentación o pon el texto entre comillas"},
		{"barra invertida", taskFile(`"C:\qa"`), "línea 10: barra invertida no válida; usa comillas simples"},
		{"tabulador al inicio", "project:\n\tname: A\n", "línea 2: el fichero tiene un tabulador o carácter no válido; usa espacios"},
		{"tabulador en un bloque", "project:\n  name: A\n\tdescription: B\n", "línea 2: el fichero tiene un tabulador o carácter no válido; usa espacios"},
		{"carácter no válido", taskFile("@autor"), "línea 10: el fichero tiene un tabulador o carácter no válido; usa espacios"},
		{"varias claves desconocidas", "project:\n  nombre: A\n  otra: B\n", `línea 2: clave desconocida "nombre"; línea 3: clave desconocida "otra"`},
		{"tipo incorrecto", "project:\n  name: [a]\n", "línea 2: el YAML no es válido"},
		{"clave repetida", "project:\n  name: A\n  name: B\n", "línea 3: el YAML no es válido"},
		{"error sin línea", "project: *ancla\n", "el YAML no es válido"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			_, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, Path: writeFile(t, tt.content)})

			assertInvalid(t, err, tt.want)
			if err.Error() != tt.want {
				t.Errorf("err = %q, se esperaba exactamente %q", err, tt.want)
			}
			if len(repo.saved) != 0 {
				t.Error("no debió guardar nada")
			}
		})
	}
}

func TestHandleRejectsCommentBesideText(t *testing.T) {
	const commentMessage = "el texto contiene un comentario; ponlo entre comillas"
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"nombre del proyecto", "project:\n  name: A # nota\n  description: B\n", "proyecto"},
		{"descripción del proyecto al final", "project:\n  name: A\n  description: B # nota", "proyecto"},
		{"nombre de etapa", `
project:
  name: A
  description: B
  stages:
    - name: C
      description: D
    - name: E # nota
      description: F
`, "etapa 2"},
		{"descripción de etapa", `
project:
  name: A
  description: B
  stages:
    - name: C
      description: D # nota
`, "etapa 1"},
		{"título de tarea", taskFile("Revisar API # urgente"), "etapa 1, tarea 2"},
		{"título que empieza con almohadilla", taskFile("#1 prioridad"), "etapa 1, tarea 2"},
		{"descripción de tarea", `
project:
  name: A
  description: B
  stages:
    - name: C
      description: D
      tasks:
        - title: E
          description: F # nota
`, "etapa 1, tarea 1"},
		{"tras un texto entre comillas", taskFile(`"Revisar" # urgente`), "etapa 1, tarea 2"},
		{"en la cabecera de un bloque", `
project:
  name: A
  description: | # nota
    B
`, "proyecto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			_, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, Path: writeFile(t, tt.content)})

			assertInvalid(t, err, tt.want+": "+commentMessage)
			if len(repo.saved) != 0 {
				t.Error("no debió guardar nada")
			}
		})
	}
}

func TestHandleKeepsHashInsideText(t *testing.T) {
	const content = `# Comentario en su propia línea.
project: # proyecto de prueba
  name: "Tienda # 1: base"
  description: 'C:\qa # sin escape'
  # Otro comentario en su propia línea.
  stages: # etapas
    - name: "*urgente* [API]"
      description: |
        Línea 1: con dos puntos # y almohadilla.
        # Esta línea también es texto.
      tasks: # pendientes
        # Comentario antes de la tarea.
        - title: Sin#comentario
          description: >-
            Texto plegado # completo.
        # Comentario después de la tarea.
`
	repo := &fakeRepo{}

	_, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, Path: writeFile(t, content)})

	if err != nil {
		t.Fatal(err)
	}
	plan := repo.saved[0]
	if plan.Project.Name != "Tienda # 1: base" || plan.Project.Description != `C:\qa # sin escape` {
		t.Errorf("proyecto = %+v", plan.Project)
	}
	stage := plan.Stages[0]
	wantDescription := "Línea 1: con dos puntos # y almohadilla.\n# Esta línea también es texto."
	if stage.Stage.Name != "*urgente* [API]" || stage.Stage.Description != wantDescription {
		t.Errorf("etapa = %+v", stage.Stage)
	}
	if stage.Tasks[0].Title != "Sin#comentario" || stage.Tasks[0].Description != "Texto plegado # completo." {
		t.Errorf("tarea = %+v", stage.Tasks[0])
	}
}

func TestHandleRejectsMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-existe.yaml")

	_, err := New(&fakeRepo{}).Handle(context.Background(), Command{Actor: kernel.Owner, Path: path})

	assertInvalid(t, err, path)
}

func TestHandlePropagatesRepositoryError(t *testing.T) {
	failure := errors.New("sin conexión")

	_, err := New(&fakeRepo{err: failure}).Handle(context.Background(), Command{Actor: kernel.Owner, Path: writeFile(t, validFile)})

	if !errors.Is(err, failure) {
		t.Errorf("err = %v", err)
	}
}

func assertInvalid(t *testing.T, err error, want string) {
	t.Helper()
	var businessErr *kernel.Error
	if !errors.As(err, &businessErr) || businessErr.Kind != kernel.KindInvalid {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("err = %q, se esperaba %q", err, want)
	}
}
