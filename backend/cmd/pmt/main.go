// Command pmt es el CLI de PMT: inicia sesión por el navegador, guarda el token
// de API del usuario en ~/.config/pmt/credenciales.yaml y levanta a PP en herdr.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"pmt/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	app, err := cli.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pmt: %s\n", err)
		os.Exit(1)
	}
	code := app.Run(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}
