package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	pgadapter "project-management-tools/internal/adapter/driven/postgres"
	mcpadapter "project-management-tools/internal/adapter/driving/mcp"
	commentapp "project-management-tools/internal/application/comment"
	issueapp "project-management-tools/internal/application/issue"
	labelapp "project-management-tools/internal/application/label"
	"project-management-tools/internal/application/notification"
	phaseapp "project-management-tools/internal/application/phase"
	projectapp "project-management-tools/internal/application/project"
	"project-management-tools/internal/domain/user"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	userEmail := os.Getenv("USER_EMAIL")
	if userEmail == "" {
		log.Fatal("USER_EMAIL is required")
	}

	if err := pgadapter.EnsureDatabase(databaseURL); err != nil {
		log.Fatalf("failed to ensure database: %v", err)
	}

	db, err := pgadapter.NewConnection(databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := pgadapter.RunMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Repositories
	userRepo := pgadapter.NewUserRepository(db)
	projectRepo := pgadapter.NewProjectRepository(db)
	phaseRepo := pgadapter.NewPhaseRepository(db)
	issueRepo := pgadapter.NewIssueRepository(db)
	labelRepo := pgadapter.NewLabelRepository(db)
	commentRepo := pgadapter.NewCommentRepository(db)

	// Resolve owner
	email, err := user.NewEmail(userEmail)
	if err != nil {
		log.Fatalf("invalid USER_EMAIL %q: %v", userEmail, err)
	}

	owner, err := userRepo.FindByEmail(context.Background(), email)
	if err != nil {
		log.Fatalf("fatal: user not found for email %s", userEmail)
	}

	// Services (MCP server has no WebSocket clients; use no-op notifier)
	noop := notification.NoopNotifier{}
	projectService := projectapp.NewService(projectRepo, noop)
	phaseService := phaseapp.NewService(phaseRepo, projectRepo, noop)
	issueService := issueapp.NewService(issueRepo, phaseRepo, projectRepo, labelRepo, noop)
	labelService := labelapp.NewService(labelRepo, projectRepo)
	commentService := commentapp.NewService(commentRepo, issueRepo)

	srv := mcpadapter.NewServer(
		owner.ID(),
		projectService,
		phaseService,
		issueService,
		labelService,
		commentService,
	)

	fmt.Fprintln(os.Stderr, "pmt mcp server started")

	if err := srv.ServeStdio(); err != nil {
		log.Fatalf("mcp server error: %v", err)
	}
}
