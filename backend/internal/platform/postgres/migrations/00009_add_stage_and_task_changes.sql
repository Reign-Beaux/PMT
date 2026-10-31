-- +goose Up
ALTER TABLE stages ADD COLUMN retired_at TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN retired_at TIMESTAMPTZ;

CREATE TABLE stage_changes (
    id           UUID PRIMARY KEY,
    stage_id     UUID NOT NULL REFERENCES stages (id) ON DELETE RESTRICT,
    version      INTEGER NOT NULL CHECK (version > 0),
    kind         TEXT NOT NULL CHECK (kind IN ('add', 'edit', 'retire')),
    name         TEXT NOT NULL CHECK (name <> ''),
    description  TEXT NOT NULL DEFAULT '',
    reason       TEXT NOT NULL CHECK (reason <> ''),
    requested_by TEXT NOT NULL CHECK (requested_by <> ''),
    changed_at   TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (stage_id, version)
);

CREATE TABLE task_changes (
    id           UUID PRIMARY KEY,
    task_id      UUID NOT NULL REFERENCES tasks (id) ON DELETE RESTRICT,
    version      INTEGER NOT NULL CHECK (version > 0),
    kind         TEXT NOT NULL CHECK (kind IN ('add', 'edit', 'retire')),
    title        TEXT NOT NULL CHECK (title <> ''),
    description  TEXT NOT NULL DEFAULT '',
    reason       TEXT NOT NULL CHECK (reason <> ''),
    requested_by TEXT NOT NULL CHECK (requested_by <> ''),
    changed_at   TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (task_id, version)
);

-- +goose StatementBegin
CREATE FUNCTION reject_change_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'los cambios son inmutables: % en %', TG_OP, TG_TABLE_NAME;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER stage_changes_immutable
    BEFORE UPDATE OR DELETE ON stage_changes
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER stage_changes_no_truncate
    BEFORE TRUNCATE ON stage_changes
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER task_changes_immutable
    BEFORE UPDATE OR DELETE ON task_changes
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER task_changes_no_truncate
    BEFORE TRUNCATE ON task_changes
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

-- +goose Down
DROP TABLE task_changes;
DROP TABLE stage_changes;
DROP FUNCTION reject_change_mutation();
ALTER TABLE tasks DROP COLUMN retired_at;
ALTER TABLE stages DROP COLUMN retired_at;
