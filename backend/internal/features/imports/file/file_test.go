package file

import (
	"errors"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"pmt/internal/kernel"
)

func TestYAMLKeepsCommentInValueNode(t *testing.T) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte("title: Revisar API # urgente\n"), &document); err != nil {
		t.Fatal(err)
	}

	_, value := field(document.Content[0], "title")

	if value.Value != "Revisar API" || value.LineComment != "# urgente" {
		t.Errorf("valor = %q, comentario = %q", value.Value, value.LineComment)
	}
}

func TestReadStagesOneStage(t *testing.T) {
	content := `
stages:
  - name: "Pagos"
    description: "Cobro con tarjeta."
    tasks:
      - title: "Tabla de pagos"
        description: "Se relaciona con pedidos."
      - title: "Cobrar"
        description: "Con tarjeta."
`
	entries, err := ParseStages(content)

	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("etapas = %d", len(entries))
	}
	if entries[0].Name != "Pagos" || entries[0].Description != "Cobro con tarjeta." {
		t.Errorf("etapa = %+v", entries[0])
	}
	if len(entries[0].Tasks) != 2 || entries[0].Tasks[1].Title != "Cobrar" {
		t.Errorf("tareas = %+v", entries[0].Tasks)
	}
}

func TestReadStagesThreeStagesInOrder(t *testing.T) {
	content := `
stages:
  - name: "Una"
    description: "Primera."
    tasks:
      - title: "T1"
        description: "D1"
  - name: "Dos"
    description: "Segunda."
  - name: "Tres"
    description: "Tercera."
    tasks:
      - title: "T2"
        description: "D2"
      - title: "T3"
        description: "D3"
`
	entries, err := ParseStages(content)

	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("etapas = %d", len(entries))
	}
	for i, want := range []struct {
		name  string
		tasks int
	}{{"Una", 1}, {"Dos", 0}, {"Tres", 2}} {
		if entries[i].Name != want.name || len(entries[i].Tasks) != want.tasks {
			t.Errorf("etapa %d = %+v con %d tareas", i+1, entries[i], len(entries[i].Tasks))
		}
	}
}

func TestReadStagesRejectsInvalidFiles(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"raíz stage",
			"stage:\n  name: A\n  description: B\n",
			"la raíz ahora es stages, con una lista de etapas",
		},
		{
			"raíz stage junto con stages",
			"stage:\n  name: A\nstages: []\n",
			"la raíz ahora es stages, con una lista de etapas",
		},
		{"stages vacía", "stages: []\n", "el fichero no tiene etapas"},
		{"stages nula", "stages:\n", "el fichero no tiene etapas"},
		{"fichero vacío", "", "el YAML está vacío"},
		{
			"etapa 2 sin descripción",
			"stages:\n  - name: A\n    description: B\n  - name: C\n",
			"etapa 2: la descripción es obligatoria",
		},
		{
			"etapa 2 sin nombre",
			"stages:\n  - name: A\n    description: B\n  - name: \" \"\n    description: C\n",
			"etapa 2: el nombre de la etapa es obligatorio",
		},
		{
			"tarea 3 de la etapa 2 sin título",
			"stages:\n  - name: A\n    description: B\n  - name: C\n    description: D\n    tasks:\n" +
				"      - title: T1\n        description: D1\n      - title: T2\n        description: D2\n      - description: D3\n",
			"etapa 2, tarea 3: el título de la tarea es obligatorio",
		},
		{
			"tarea sin descripción",
			"stages:\n  - name: A\n    description: B\n    tasks:\n      - title: T\n",
			"etapa 1, tarea 1: la descripción es obligatoria",
		},
		{
			"clave desconocida",
			"stages:\n  - name: A\n    description: B\n    extra: 1\n",
			`línea 4: clave desconocida "extra"`,
		},
		{
			"YAML inválido",
			"stages:\n  - name: Pagos: urgentes\n    description: B\n",
			`línea 2: el texto contiene ":"; ponlo entre comillas`,
		},
		{
			"stages con un escalar",
			"stages: hola\n",
			"línea 1: el YAML no es válido",
		},
		{
			"comentario en la etapa 2",
			"stages:\n  - name: A\n    description: B\n  - name: C # nota\n    description: D\n",
			"etapa 2: el texto contiene un comentario; ponlo entre comillas",
		},
		{
			"comentario en tarea de la etapa 2",
			"stages:\n  - name: A\n    description: B\n  - name: C\n    description: D\n    tasks:\n      - title: T # nota\n        description: E\n",
			"etapa 2, tarea 1: el texto contiene un comentario; ponlo entre comillas",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseStages(tt.content)

			if err == nil || err.Error() != tt.want {
				t.Errorf("err = %v, se esperaba %q", err, tt.want)
			}
		})
	}
}

func TestReadStagesAcceptsSpecialTextsAndAliases(t *testing.T) {
	content := `
stages:
  - name: "Pagos: tarjeta y más"
    description: &desc "Descripción con acentos y \"comillas\"."
    tasks:
      - title: "Tarea 1"
        description: *desc
`
	entries, err := ParseStages(content)

	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Name != "Pagos: tarjeta y más" {
		t.Errorf("nombre = %q", entries[0].Name)
	}
	want := `Descripción con acentos y "comillas".`
	if entries[0].Description != want || entries[0].Tasks[0].Description != want {
		t.Errorf("descripciones = %q, %q", entries[0].Description, entries[0].Tasks[0].Description)
	}
}

func TestParseRejectsEmptyContent(t *testing.T) {
	for _, content := range []string{"", "   \n\t\n", "# solo un comentario\n"} {
		_, projectErr := ParseProject(content)
		_, stagesErr := ParseStages(content)

		for _, err := range []error{projectErr, stagesErr} {
			assertInvalid(t, err, "el YAML está vacío")
		}
	}
}

func TestParseRejectsContentOverLimit(t *testing.T) {
	head := "stages:\n  - name: \"A\"\n    description: |\n"
	content := head + strings.Repeat("      texto de relleno\n", MaxContentBytes/20)

	_, stagesErr := ParseStages(content)
	_, projectErr := ParseProject(content)

	for _, err := range []error{stagesErr, projectErr} {
		assertInvalid(t, err, "el tope es 1 MiB")
	}
}

func TestParseAcceptsContentAtLimit(t *testing.T) {
	head := "stages:\n  - name: \"A\"\n    description: \"B\"\n"
	content := head + strings.Repeat("#", MaxContentBytes-len(head))

	entries, err := ParseStages(content)

	if err != nil || len(entries) != 1 {
		t.Errorf("etapas = %+v, err = %v", entries, err)
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
