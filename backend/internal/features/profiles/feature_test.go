package profiles

import (
	"strings"
	"testing"

	"pmt/internal/features/profiles/catalog"
	"pmt/internal/kernel"
)

// Claude Code solo muestra los primeros ~4 000 caracteres de las instrucciones.
const instructionsLimit = 4000

func TestInstructionsCortasConPerfil(t *testing.T) {
	for _, e := range catalog.List() {
		text, ok := Instructions(kernel.Actor{Name: "ana", Profile: e.Profile})
		if !ok {
			t.Fatalf("%s sin instrucciones", e.Profile)
		}
		if len(text) >= instructionsLimit {
			t.Errorf("%s: %d caracteres", e.Profile, len(text))
		}
		for _, want := range []string{"perfil " + string(e.Profile) + " (" + e.Description + ")", "get_profile", catalog.EnvFile} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: falta %q en %q", e.Profile, want, text)
			}
		}
		if strings.Index(text, "get_profile") > strings.Index(text, catalog.EnvFile) {
			t.Errorf("%s: get_profile debe ir antes que %s", e.Profile, catalog.EnvFile)
		}
	}
}

func TestInstructionsSinPerfilNiAgente(t *testing.T) {
	for name, actor := range map[string]kernel.Actor{
		"usuario sin perfil": {Name: "ana", Profile: kernel.ProfileUser},
		"perfil vacío":       {Name: "ana"},
		"token de agente":    {Name: "dbp-tl", Profile: kernel.ProfileTL, Agent: true},
	} {
		if text, ok := Instructions(actor); ok || text != "" {
			t.Errorf("%s: %q", name, text)
		}
	}
}
