-- +goose Up
ALTER TABLE stages
    ADD COLUMN status TEXT NOT NULL DEFAULT 'planning' CHECK (status IN (
        'planning',
        'stand_by',
        'in_dev',
        'done',
        'stopped',
        'cancelled'
    ));

-- +goose Down
ALTER TABLE stages
    DROP COLUMN status;
