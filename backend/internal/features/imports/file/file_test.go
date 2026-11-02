package file

import (
	"testing"

	"gopkg.in/yaml.v3"
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
