-- +goose Up
CREATE TABLE stages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    name        TEXT NOT NULL CHECK (name <> ''),
    description TEXT NOT NULL DEFAULT '',
    position    INTEGER NOT NULL CHECK (position > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stages_project_position_key
        UNIQUE (project_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- +goose Down
DROP TABLE stages;
