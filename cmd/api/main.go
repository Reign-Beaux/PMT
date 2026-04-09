package main

import (
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	pgadapter "project-management-tools/internal/adapter/driven/postgres"
	"project-management-tools/internal/adapter/driving/httpserver"
	"project-management-tools/internal/adapter/driving/httpserver/handler"
	issueapp "project-management-tools/internal/application/issue"
	phaseapp "project-management-tools/internal/application/phase"
	projectapp "project-management-tools/internal/application/project"
	"project-management-tools/internal/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	cfg := config.Load()

	if err := pgadapter.EnsureDatabase(cfg.DatabaseURL); err != nil {
		log.Fatalf("failed to ensure database: %v", err)
	}

	db, err := pgadapter.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := pgadapter.RunMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Repositories
	projectRepo := pgadapter.NewProjectRepository(db)
	phaseRepo := pgadapter.NewPhaseRepository(db)
	issueRepo := pgadapter.NewIssueRepository(db)

	// Services
	projectService := projectapp.NewService(projectRepo)
	phaseService := phaseapp.NewService(phaseRepo, projectRepo)
	issueService := issueapp.NewService(issueRepo, phaseRepo, projectRepo)

	// Handlers
	projectHandler := handler.NewProjectHandler(projectService)
	phaseHandler := handler.NewPhaseHandler(phaseService)
	issueHandler := handler.NewIssueHandler(issueService)

	router := httpserver.NewRouter(projectHandler, phaseHandler, issueHandler)

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
