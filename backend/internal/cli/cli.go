// Package cli es el CLI pmt: inicia sesión en un servidor PMT por el navegador y
// guarda el token de API del usuario en ~/.config/pmt/credenciales.yaml.
//
// Usa flag de la biblioteca estándar: con cuatro subcomandos y una opción no
// hace falta una librería. cobra solo está en el módulo como dependencia
// indirecta de sqlc y agrandaría el binario.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Version es la versión del binario. La fija el compilado con
// -ldflags "-X pmt/internal/cli.Version=<versión>".
var Version = "dev"

// App es el CLI con sus dependencias, para poder probarlo.
type App struct {
	Stdout   io.Writer
	Stderr   io.Writer
	Home     string
	HTTP     *http.Client
	Hostname func() (string, error)
	// OpenBrowser abre una URL en el navegador. Devuelve error si no pudo.
	OpenBrowser func(url string) error
	// Sleep espera entre sondeos; termina antes si ctx se cancela.
	Sleep func(ctx context.Context, d time.Duration) error
	Now   func() time.Time
}

// New crea el CLI real: la carpeta personal del usuario, el navegador del sistema
// y un cliente HTTP con timeout.
func New() (*App, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("no encuentro tu carpeta personal: %w", err)
	}
	return &App{
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Home:        home,
		HTTP:        &http.Client{Timeout: 20 * time.Second},
		Hostname:    os.Hostname,
		OpenBrowser: openBrowser,
		Sleep:       sleep,
		Now:         time.Now,
	}, nil
}

const usage = `pmt: el CLI de PMT.

Uso:
  pmt login [--servidor URL] [--sin-navegador]   inicia sesión por el navegador
  pmt logout                                    revoca el token y cierra la sesión
  pmt whoami                                    muestra tu usuario y el servidor
  pmt version                                   muestra la versión

Por omisión el servidor es ` + DefaultServer + `; el que uses en login queda guardado.
`

// Run ejecuta un subcomando y devuelve el código de salida.
func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(a.Stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "login":
		err = a.login(ctx, args[1:])
	case "logout":
		err = noArgs(ctx, "logout", args[1:], a.logout)
	case "whoami":
		err = noArgs(ctx, "whoami", args[1:], a.whoami)
	case "version", "--version", "-v":
		fmt.Fprintf(a.Stdout, "pmt %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
	case "help", "--help", "-h", "ayuda":
		fmt.Fprint(a.Stdout, usage)
	default:
		fmt.Fprintf(a.Stderr, "pmt: comando desconocido %q\n\n%s", args[0], usage)
		return 2
	}
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	var usageErr usageError
	if errors.As(err, &usageErr) {
		fmt.Fprintf(a.Stderr, "pmt: %s\n", usageErr.msg)
		return 2
	}
	if err != nil {
		fmt.Fprintf(a.Stderr, "pmt: %s\n", err)
		return 1
	}
	return 0
}

// usageError es un error de uso: sale con 2.
type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func noArgs(ctx context.Context, name string, args []string, run func(context.Context) error) error {
	if len(args) > 0 {
		return usageError{fmt.Sprintf("%s no recibe argumentos", name)}
	}
	return run(ctx)
}

func (a *App) client(server, token string) client {
	return client{http: a.HTTP, server: server, token: token, agent: "pmt/" + Version}
}

// login pide un código, abre el navegador y sondea hasta que la persona lo
// autorice, lo cancele o caduque. Guarda las credenciales con el servidor.
func (a *App) login(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	serverFlag := fs.String("servidor", "", "URL del servidor PMT (se guarda para la próxima vez)")
	noBrowser := fs.Bool("sin-navegador", false, "no abrir el navegador; solo mostrar la URL")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return usageError{err.Error()}
	}
	if fs.NArg() > 0 {
		return usageError{"login solo recibe --servidor y --sin-navegador"}
	}

	previous, prevErr := loadCredentials(a.Home)
	if prevErr != nil && !errors.Is(prevErr, errNoCredentials) {
		return prevErr
	}
	server := DefaultServer
	switch {
	case *serverFlag != "":
		server = *serverFlag
	case previous.Server != "":
		server = previous.Server
	}
	server, err := normalizeServer(server)
	if err != nil {
		return err
	}

	machine, err := a.Hostname()
	if err != nil || strings.TrimSpace(machine) == "" {
		machine = "sin-nombre"
	}
	if len([]rune(machine)) > 60 {
		machine = string([]rune(machine)[:60])
	}

	api := a.client(server, "")
	var start loginStart
	if err := api.do(ctx, http.MethodPost, "/api/cli/login", map[string]string{"machine": machine}, &start); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "Código: %s\n", start.Code)
	fmt.Fprintf(a.Stdout, "Comprueba que el navegador muestre el mismo código y autoriza.\n")
	opened := false
	if !*noBrowser {
		opened = a.OpenBrowser(start.URL) == nil
	}
	if opened {
		fmt.Fprintf(a.Stdout, "Se abrió el navegador. Si no, abre: %s\n", start.URL)
	} else {
		fmt.Fprintf(a.Stdout, "Abre en un navegador (puede ser en otro equipo): %s\n", start.URL)
	}
	fmt.Fprintln(a.Stdout, "Esperando la autorización…")

	poll, err := a.waitAuthorization(ctx, api, start)
	if err != nil {
		return err
	}
	creds := Credentials{Server: server, Username: poll.Username, Token: poll.Token, TokenID: poll.TokenID}
	if err := saveCredentials(a.Home, creds); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "Sesión iniciada como %s en %s.\n", poll.Username, server)

	// El token anterior, si lo había, ya no se usa: se revoca para no dejarlo vivo.
	if previous.Token != "" && previous.TokenID != "" && previous.TokenID != poll.TokenID {
		old := a.client(previous.Server, previous.Token)
		if err := old.do(ctx, http.MethodDelete, "/api/tokens/"+previous.TokenID, nil, nil); err != nil && !isStatus(err, http.StatusUnauthorized) && !isStatus(err, http.StatusNotFound) {
			fmt.Fprintf(a.Stderr, "Aviso: no se pudo revocar el token anterior en %s: %s. Revócalo en la página Cuenta.\n", previous.Server, err)
		}
	}
	return nil
}

// waitAuthorization sondea cada interval segundos hasta un estado final o hasta
// que caduque el código. Un fallo de red se reintenta.
func (a *App) waitAuthorization(ctx context.Context, api client, start loginStart) (loginPoll, error) {
	interval := time.Duration(start.Interval) * time.Second
	if interval < time.Second {
		interval = 2 * time.Second
	}
	warned := false
	for {
		if err := a.Sleep(ctx, interval); err != nil {
			return loginPoll{}, errors.New("cancelado")
		}
		var poll loginPoll
		err := api.do(ctx, http.MethodPost, "/api/cli/login/poll", map[string]string{"poll_secret": start.PollSecret}, &poll)
		switch {
		case isStatus(err, http.StatusNotFound):
			return loginPoll{}, errors.New("el código caducó; vuelve a correr «pmt login»")
		case err != nil:
			var apiErr *apiError
			if errors.As(err, &apiErr) {
				return loginPoll{}, err
			}
			if !warned {
				fmt.Fprintf(a.Stderr, "Aviso: %s; reintentando…\n", err)
				warned = true
			}
			if !start.ExpiresAt.IsZero() && a.Now().After(start.ExpiresAt) {
				return loginPoll{}, errors.New("el código caducó; vuelve a correr «pmt login»")
			}
			continue
		}
		switch poll.Status {
		case "pending":
			continue
		case "authorized":
			if poll.Token == "" {
				return loginPoll{}, errors.New("el servidor autorizó pero no entregó el token")
			}
			return poll, nil
		case "cancelled":
			return loginPoll{}, errors.New("se canceló el inicio de sesión")
		case "expired":
			return loginPoll{}, errors.New("el código caducó; vuelve a correr «pmt login»")
		default:
			return loginPoll{}, fmt.Errorf("estado desconocido %q", poll.Status)
		}
	}
}

// logout revoca el token en el servidor y borra las credenciales. Si el token ya
// no vale, solo borra el archivo; si no hay conexión, no borra nada.
func (a *App) logout(ctx context.Context) error {
	creds, err := loadCredentials(a.Home)
	if errors.Is(err, errNoCredentials) {
		fmt.Fprintln(a.Stdout, "No había sesión iniciada.")
		return nil
	}
	if err != nil {
		return err
	}
	if creds.TokenID != "" {
		err = a.client(creds.Server, creds.Token).do(ctx, http.MethodDelete, "/api/tokens/"+creds.TokenID, nil, nil)
		if err != nil && !isStatus(err, http.StatusUnauthorized) && !isStatus(err, http.StatusNotFound) {
			return fmt.Errorf("no se pudo revocar el token: %w; la sesión sigue guardada", err)
		}
	}
	if err := removeCredentials(a.Home); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "Sesión cerrada en %s.\n", creds.Server)
	return nil
}

// whoami muestra el usuario del token y el servidor.
func (a *App) whoami(ctx context.Context) error {
	creds, err := loadCredentials(a.Home)
	if err != nil {
		return err
	}
	var user me
	err = a.client(creds.Server, creds.Token).do(ctx, http.MethodGet, "/api/auth/me", nil, &user)
	if isStatus(err, http.StatusUnauthorized) {
		return fmt.Errorf("tu sesión en %s ya no es válida; corre «pmt login»", creds.Server)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "%s (%s) en %s\n", user.Username, user.Name, creds.Server)
	return nil
}

// openBrowser abre la URL con xdg-open en Linux u open en macOS, sin esperar.
// En Linux sin sesión gráfica no lo intenta.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return errors.New("sin sesión gráfica")
		}
		cmd = exec.Command("xdg-open", url)
	default:
		return errors.New("sistema sin navegador conocido")
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
