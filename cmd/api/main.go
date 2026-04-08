package main

import (
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	pgadapter "project-management-tools/internal/adapter/driven/postgres"
	"project-management-tools/internal/adapter/driving/httpserver"
	"project-management-tools/internal/adapter/driving/httpserver/handler"
	projectapp "project-management-tools/internal/application/project"
	"project-management-tools/internal/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	cfg := config.Load()

	db, err := pgadapter.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := pgadapter.RunMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Projects
	projectRepo := pgadapter.NewProjectRepository(db)
	projectService := projectapp.NewService(projectRepo)
	projectHandler := handler.NewProjectHandler(projectService)

	router := httpserver.NewRouter(projectHandler)

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
