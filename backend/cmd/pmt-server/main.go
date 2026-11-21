package main

import (
	"log"
	"os"

	"pmt/internal/app"
	"pmt/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "agent" {
		err = app.RunAgent(cfg, os.Args[2:], os.Stdout)
	} else {
		err = app.Run(cfg)
	}
	if err != nil {
		log.Fatal(err)
	}
}
