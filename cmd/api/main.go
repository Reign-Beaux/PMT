package main

import (
	"log"
	"net/http"
	"time"

	"project-management-tools/internal/adapter/driven/postgres"
	"project-management-tools/internal/adapter/driving/httpserver"
	"project-management-tools/internal/config"
)

func main() {
	cfg := config.Load()

	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := postgres.RunMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	router := httpserver.NewRouter()

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("server running on port %s", cfg.Port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
