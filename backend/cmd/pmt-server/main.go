package main

import (
	"log"
	"os"

	"pmt/internal/app"
)

func main() {
	run := app.Run
	if len(os.Args) > 1 && os.Args[1] == "agent" {
		run = func() error { return app.RunAgent(os.Args[2:], os.Stdout) }
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
