-- +goose Up
CREATE TABLE bugs (
    id                 UUID PRIMARY KEY,
    project_id         UUID NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    title              TEXT NOT NULL CHECK (title <> ''),
    description        TEXT NOT NULL DEFAULT '',
    steps_to_reproduce TEXT NOT NULL CHECK (steps_to_reproduce <> ''),
    product_version    TEXT NOT NULL CHECK (product_version <> ''),
    status             TEXT NOT NULL DEFAULT 'reported' CHECK (status IN (
        'reported', 'confirmed', 'in_progress', 'in_review',
        'changes_requested', 'in_qa', 'done', 'rejected'
    )),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX bugs_project_id_idx ON bugs (project_id);

-- +goose Down
DROP TABLE bugs;
