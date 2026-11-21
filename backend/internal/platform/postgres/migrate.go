package postgres

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate aplica las migraciones embebidas que falten. Se llama al arrancar.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("cerrar conexión de migraciones", "error", err)
		}
	}()
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(database.DialectPostgres, db, files)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}
