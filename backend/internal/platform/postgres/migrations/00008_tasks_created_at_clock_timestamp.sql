-- +goose Up
-- now() es fija dentro de una transacción; clock_timestamp() conserva el orden de inserción.
ALTER TABLE tasks
    ALTER COLUMN created_at SET DEFAULT clock_timestamp();

-- +goose Down
ALTER TABLE tasks
    ALTER COLUMN created_at SET DEFAULT now();
