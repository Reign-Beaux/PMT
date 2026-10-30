-- +goose Up
ALTER TABLE tasks
    DROP CONSTRAINT tasks_status_check,
    ADD CONSTRAINT tasks_status_check CHECK (status IN (
        'backlog',
        'ready',
        'in_progress',
        'in_review',
        'changes_requested',
        'in_qa',
        'pending_close',
        'done',
        'cancelled'
    ));

-- +goose Down
-- Falla si existen tareas en 'cancelled', porque el CHECK anterior no las admite.
ALTER TABLE tasks
    DROP CONSTRAINT tasks_status_check,
    ADD CONSTRAINT tasks_status_check CHECK (status IN (
        'backlog',
        'ready',
        'in_progress',
        'in_review',
        'changes_requested',
        'in_qa',
        'pending_close',
        'done'
    ));
