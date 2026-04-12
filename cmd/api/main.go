package main

import (
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	pgadapter "project-management-tools/internal/adapter/driven/postgres"
	"project-management-tools/internal/adapter/driving/httpserver"
	"project-management-tools/internal/adapter/driving/httpserver/handler"
	"project-management-tools/internal/adapter/driving/httpserver/ws"
	commentapp "project-management-tools/internal/application/comment"
	issueapp "project-management-tools/internal/application/issue"
	labelapp "project-management-tools/internal/application/label"
	phaseapp "project-management-tools/internal/application/phase"
	projectapp "project-management-tools/internal/application/project"
	userapp "project-management-tools/internal/application/user"
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

	jwtSecret := []byte(cfg.JWTSecret)

	// WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// Repositories
	userRepo := pgadapter.NewUserRepository(db)
	tokenRepo := pgadapter.NewTokenRepository(db)
	projectRepo := pgadapter.NewProjectRepository(db)
	phaseRepo := pgadapter.NewPhaseRepository(db)
	issueRepo := pgadapter.NewIssueRepository(db)
	labelRepo := pgadapter.NewLabelRepository(db)
	commentRepo := pgadapter.NewCommentRepository(db)

	// Services
	userService := userapp.NewService(userRepo, tokenRepo)
	projectService := projectapp.NewService(projectRepo, hub)
	phaseService := phaseapp.NewService(phaseRepo, projectRepo, hub)
	issueService := issueapp.NewService(issueRepo, phaseRepo, projectRepo, labelRepo, hub)
	labelService := labelapp.NewService(labelRepo, projectRepo)
	commentService := commentapp.NewService(commentRepo, issueRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(userService, jwtSecret)
	projectHandler := handler.NewProjectHandler(projectService)
	phaseHandler := handler.NewPhaseHandler(phaseService)
	issueHandler := handler.NewIssueHandler(issueService)
	labelHandler := handler.NewLabelHandler(labelService, issueService)
	commentHandler := handler.NewCommentHandler(commentService)
	wsHandler := ws.NewHandler(hub, jwtSecret)

	router := httpserver.NewRouter(
		authHandler,
		projectHandler,
		phaseHandler,
		issueHandler,
		labelHandler,
		commentHandler,
		wsHandler,
		jwtSecret,
	)

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
