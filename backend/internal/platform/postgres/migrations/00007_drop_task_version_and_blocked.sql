-- +goose Up
ALTER TABLE tasks
    DROP COLUMN version,
    DROP COLUMN blocked;

-- +goose Down
-- Restaura las columnas con sus valores por defecto; los valores anteriores no se recuperan.
ALTER TABLE tasks
    ADD COLUMN blocked BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
