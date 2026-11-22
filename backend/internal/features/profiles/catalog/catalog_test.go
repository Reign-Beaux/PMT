package catalog

import (
	"strings"
	"testing"

	"pmt/internal/kernel"
)

func TestLosNuevePerfilesSeSirven(t *testing.T) {
	if len(entries) != 9 {
		t.Fatalf("perfiles = %d, quiero 9", len(entries))
	}
	for _, e := range List() {
		t.Run(e.Name, func(t *testing.T) {
			text, ok := Served(e.Profile)
			if !ok || text == "" {
				t.Fatal("sin texto servido")
			}
			if _, err := kernel.ParseProfile(e.Name); err != nil {
				t.Errorf("el nombre no se puede declarar: %v", err)
			}
			if !strings.Contains(text, "# Eres ") {
				t.Errorf("no trae el perfil: %.80q", text)
			}
			for _, rest := range []string{"{{", "}}", "[[", "]]", "||"} {
				if strings.Contains(text, rest) {
					t.Errorf("queda %q sin resolver", rest)
				}
			}
			if !strings.Contains(text, EnvFile) {
				t.Errorf("no menciona %s", EnvFile)
			}
			if strings.Contains(text, ".sh") {
				t.Errorf("menciona un script del kit")
			}
		})
	}
}

func TestLoComunAbrePidiendoElEntorno(t *testing.T) {
	for _, name := range []string{"comun.md", "pc-comun.md"} {
		text := strings.TrimSpace(render(read(name)))
		if !strings.HasPrefix(text, "Al empezar la sesión, antes de cualquier otra cosa, lee `"+EnvFile+"`") {
			t.Errorf("%s abre con %.80q", name, text)
		}
	}
}

func TestPCNoRecibeLasReglasDePMT(t *testing.T) {
	pc, _ := Served(kernel.ProfilePC)
	tl, _ := Served(kernel.ProfileTL)
	if strings.Contains(pc, "Reglas:") || !strings.Contains(tl, "Reglas:") {
		t.Error("solo PC debe quedar sin comun.md")
	}
	if !strings.HasPrefix(tl, "# Eres TL") {
		t.Errorf("el perfil va antes que lo común: %.40q", tl)
	}
}

func TestUserNoTienePerfil(t *testing.T) {
	if _, ok := Served(kernel.ProfileUser); ok {
		t.Error("User no debe tener texto")
	}
	if _, ok := Served(""); ok {
		t.Error("sin perfil no debe haber texto")
	}
}

func TestByName(t *testing.T) {
	if e, ok := ByName("DevBackend"); !ok || e.Profile != kernel.ProfileDevBackend {
		t.Errorf("DevBackend = %+v, %t", e, ok)
	}
	for _, name := range []string{"user", "", "devback", "xx"} {
		if _, ok := ByName(name); ok {
			t.Errorf("%q no debe existir", name)
		}
	}
}
