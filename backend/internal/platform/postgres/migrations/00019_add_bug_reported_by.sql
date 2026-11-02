-- +goose Up
ALTER TABLE bugs ADD COLUMN reported_by TEXT;

-- +goose Down
ALTER TABLE bugs DROP COLUMN reported_by;
