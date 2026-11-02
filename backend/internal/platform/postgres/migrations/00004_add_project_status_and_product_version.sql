-- +goose Up
ALTER TABLE projects
    ADD COLUMN status TEXT NOT NULL DEFAULT 'planning' CHECK (status IN (
        'planning',
        'stand_by',
        'in_dev',
        'working',
        'stopped',
        'cancelled'
    )),
    ADD COLUMN product_version TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE projects
    DROP COLUMN product_version,
    DROP COLUMN status;
