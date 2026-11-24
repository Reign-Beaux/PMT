package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ctrlC es un stdin que no responde: al primer Read cancela el contexto, como
// Ctrl+C mientras pmt espera una respuesta, y se queda bloqueado.
type ctrlC struct {
	cancel  context.CancelFunc
	release chan struct{}
}

func (c *ctrlC) Read([]byte) (int, error) {
	c.cancel()
	<-c.release
	return 0, errors.New("cerrado")
}

func runWithCtrlC(t *testing.T, h *ppHarness, args ...string) int {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stdin := &ctrlC{cancel: cancel, release: make(chan struct{})}
	t.Cleanup(func() { close(stdin.release) })
	h.app.Stdin = stdin
	return h.app.Run(ctx, args)
}

func TestPrompterLineasYFin(t *testing.T) {
	p := newPrompter(context.Background(), strings.NewReader(" uno \ndos"))
	for _, want := range []string{"uno", "dos"} {
		if got, err := p.readLine(); err != nil || got != want {
			t.Fatalf("readLine = %q, %v; quería %q", got, err, want)
		}
	}
	for range 2 {
		if _, err := p.readLine(); !errors.Is(err, errNoInput) {
			t.Fatalf("tras el fin: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newPrompter(ctx, strings.NewReader("x\n")).readLine(); !errors.Is(err, errInterrupted) {
		t.Fatalf("con el contexto cancelado: %v", err)
	}
}

func TestPPCtrlCEnLaPreguntaNoGuarda(t *testing.T) {
	srv := newPPServer(t)
	h := newPPHarness(t, srv)
	if code := runWithCtrlC(t, h, "pp"); code != 130 || !strings.Contains(h.errOut.String(), "Ctrl+C: no guardé nada") {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if _, err := os.Stat(filepath.Join(h.root, pmtDir)); !os.IsNotExist(err) {
		t.Fatalf("no debía crear .pmt/: %v", err)
	}
	if len(h.hd.runs) != 0 || len(h.hd.panes) != 0 {
		t.Fatal("no debía lanzar")
	}
}

func TestPPCtrlDEnLaPreguntaNoGuarda(t *testing.T) {
	srv := newPPServer(t)
	h := newPPHarness(t, srv)
	h.app.Stdin = strings.NewReader("\nclaude\n") // falta modelo y effort
	if code := h.run("pp"); code != 1 || !strings.Contains(h.errOut.String(), "Ctrl+D") {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if _, err := os.Stat(filepath.Join(h.root, pmtDir)); !os.IsNotExist(err) {
		t.Fatalf("no debía crear .pmt/: %v", err)
	}
}

// equipoSinDescripcion deja la config de un proyecto al que le falta la
// descripción y devuelve su contenido.
func equipoSinDescripcion(t *testing.T) (*ppHarness, []byte) {
	t.Helper()
	cfg := teamConfig()
	cfg.Proyecto.Descripcion = ""
	h := newTeamHarness(t, cfg)
	raw, err := os.ReadFile(filepath.Join(h.root, pmtDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return h, raw
}

func TestEquipoCtrlCEnDescripcionNoGuarda(t *testing.T) {
	h, before := equipoSinDescripcion(t)
	if code := runWithCtrlC(t, h, "equipo"); code != 130 || !strings.Contains(h.errOut.String(), "no guardé nada") {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if !strings.Contains(h.out.String(), "Descripción del proyecto") {
		t.Fatalf("no llegó a preguntar:\n%s", h.out)
	}
	after, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "config.yaml"))
	if string(after) != string(before) {
		t.Fatalf("cambió config.yaml:\n%s", after)
	}
	if len(h.hd.runs) != 0 {
		t.Fatal("no debía lanzar")
	}
}

func TestEquipoCtrlDAMitadNoGuarda(t *testing.T) {
	cfg := teamConfig()
	cfg.Proyecto.Descripcion = ""
	cfg.Puertos = PortsConfig{}
	h := newTeamHarness(t, cfg)
	before, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "config.yaml"))
	// Descripción y puerto del backend; la entrada se acaba en el del frontend.
	h.app.Stdin = strings.NewReader("Tiene backend/.\n9000\n")
	if code := h.run("equipo"); code != 1 || !strings.Contains(h.errOut.String(), "no guardé nada") {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	after, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "config.yaml"))
	if string(after) != string(before) {
		t.Fatalf("cambió config.yaml:\n%s", after)
	}
	if len(h.hd.runs) != 0 {
		t.Fatal("no debía lanzar")
	}
}
