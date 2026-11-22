// Package catalog guarda el texto de los perfiles de los agentes y lo prepara para
// servirlo. Es la única fuente: el kit de agentes/ lee estos mismos archivos.
//
// Los datos de la máquina (ruta del proyecto, base, puertos, comandos del equipo)
// no los conoce el servidor. Donde el texto los usa hay un tramo con dos
// versiones, [[local||servido]]: el kit toma la local y rellena sus {{MARCADORES}};
// el servidor toma la servida, que remite a .pmt/entorno.md del proyecto.
package catalog

import (
	"embed"
	"regexp"
	"strings"

	"pmt/internal/kernel"
)

//go:embed comun.md pc-comun.md perfiles/*.md
var files embed.FS

// Entry es un perfil del catálogo.
type Entry struct {
	// Name es como lo declara una sesión: ?profile=<name> o X-PMT-Profile.
	Name        string         `json:"name"`
	Profile     kernel.Profile `json:"profile"`
	Description string         `json:"description"`
	file        string
}

// entries siguen el orden del flujo. file es el nombre del archivo en perfiles/,
// el mismo que el agente en el kit.
var entries = []Entry{
	{Name: "pp", Profile: kernel.ProfilePP, Description: "Project Planning", file: "pp"},
	{Name: "pc", Profile: kernel.ProfilePC, Description: "Project Coordinator", file: "pc"},
	{Name: "tl", Profile: kernel.ProfileTL, Description: "Tech Lead", file: "tl"},
	{Name: "dp", Profile: kernel.ProfileDP, Description: "Development Planning", file: "dp"},
	{Name: "devbackend", Profile: kernel.ProfileDevBackend, Description: "Developer Backend", file: "devback"},
	{Name: "devfrontend", Profile: kernel.ProfileDevFrontend, Description: "Developer Frontend", file: "devfront"},
	{Name: "cr", Profile: kernel.ProfileCR, Description: "Code Reviewer", file: "cr"},
	{Name: "qap", Profile: kernel.ProfileQAP, Description: "Quality Assurance Planning", file: "qap"},
	{Name: "qa", Profile: kernel.ProfileQA, Description: "Quality Assurance", file: "qa"},
}

// EnvFile es el archivo local del proyecto al que remite la versión servida.
const EnvFile = ".pmt/entorno.md"

var span = regexp.MustCompile(`(?s)\[\[(.*?)\|\|(.*?)\]\]`)

// served guarda el texto servido de cada perfil, armado una vez al arrancar.
var served = func() map[kernel.Profile]string {
	out := make(map[kernel.Profile]string, len(entries))
	for _, e := range entries {
		out[e.Profile] = render(read("perfiles/"+e.file+".md") + read(common(e.Profile)))
	}
	return out
}()

// common es lo que se agrega al perfil, como lo junta el kit: PC solo lee herdr,
// así que recibe pc-comun.md, sin reglas de PMT ni avisos; los demás, comun.md.
func common(profile kernel.Profile) string {
	if profile == kernel.ProfilePC {
		return "pc-comun.md"
	}
	return "comun.md"
}

func read(name string) string {
	raw, err := files.ReadFile(name)
	if err != nil {
		panic("catalog: falta " + name)
	}
	return string(raw)
}

// render toma la versión servida de cada tramo [[local||servido]].
func render(text string) string {
	return span.ReplaceAllString(text, "$2")
}

// List devuelve los perfiles del catálogo.
func List() []Entry {
	return append([]Entry(nil), entries...)
}

// ByName busca un perfil por su nombre de sesión, sin distinguir mayúsculas.
func ByName(name string) (Entry, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, e := range entries {
		if e.Name == name {
			return e, true
		}
	}
	return Entry{}, false
}

// ByProfile busca un perfil por su valor de kernel. User no está en el catálogo.
func ByProfile(profile kernel.Profile) (Entry, bool) {
	for _, e := range entries {
		if e.Profile == profile {
			return e, true
		}
	}
	return Entry{}, false
}

// Served devuelve el texto servido de un perfil: el perfil más lo común. User, o
// un perfil sin texto, no tiene: devuelve false.
func Served(profile kernel.Profile) (string, bool) {
	text, ok := served[profile]
	return text, ok
}
