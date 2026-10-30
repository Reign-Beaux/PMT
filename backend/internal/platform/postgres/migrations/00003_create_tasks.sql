-- +goose Up
CREATE TABLE tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_id    UUID NOT NULL REFERENCES stages (id) ON DELETE RESTRICT,
    title       TEXT NOT NULL CHECK (title <> ''),
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'backlog' CHECK (status IN (
        'backlog',
        'ready',
        'in_progress',
        'in_review',
        'changes_requested',
        'in_qa',
        'pending_close',
        'done'
    )),
    blocked     BOOLEAN NOT NULL DEFAULT false,
    version     BIGINT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX tasks_stage_id_idx ON tasks (stage_id);

-- +goose Down
DROP TABLE tasks;
