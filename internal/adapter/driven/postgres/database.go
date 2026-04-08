package postgres

import (
	"fmt"
	"net/url"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// EnsureDatabase creates the target database if it does not exist.
// Equivalent to EF Core's Database.EnsureCreated() — connects to the
// postgres system database first, then creates the target DB if missing.
func EnsureDatabase(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("invalid DSN: %w", err)
	}

	dbName := u.Path[1:] // strip leading "/"

	// Connect to the default "postgres" database to run CREATE DATABASE
	u.Path = "/postgres"
	adminDSN := u.String()

	db, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres system database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	defer sqlDB.Close()

	var exists bool
	if err := db.Raw("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)", dbName).
		Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		// Database names cannot be parameterized — safe here because dbName
		// comes from our own config, not user input.
		if err := db.Exec(fmt.Sprintf("CREATE DATABASE %q", dbName)).Error; err != nil {
			return fmt.Errorf("failed to create database %q: %w", dbName, err)
		}
	}

	return nil
}
