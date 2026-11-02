-- +goose Up
-- Sin llave foránea a su entidad: el historial de status no impide eliminar en
-- planning y sobrevive al elemento eliminado, consultable por su ID.
CREATE TABLE project_status_changes (
    id            UUID PRIMARY KEY,
    project_id    UUID NOT NULL,
    from_status   TEXT NOT NULL,
    to_status     TEXT NOT NULL,
    actor_name    TEXT NOT NULL,
    actor_profile TEXT NOT NULL,
    reason        TEXT NOT NULL DEFAULT '',
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX project_status_changes_entity_idx ON project_status_changes (project_id, changed_at);

CREATE TABLE stage_status_changes (
    id            UUID PRIMARY KEY,
    stage_id      UUID NOT NULL,
    from_status   TEXT NOT NULL,
    to_status     TEXT NOT NULL,
    actor_name    TEXT NOT NULL,
    actor_profile TEXT NOT NULL,
    reason        TEXT NOT NULL DEFAULT '',
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX stage_status_changes_entity_idx ON stage_status_changes (stage_id, changed_at);

CREATE TABLE task_status_changes (
    id            UUID PRIMARY KEY,
    task_id       UUID NOT NULL,
    from_status   TEXT NOT NULL,
    to_status     TEXT NOT NULL,
    actor_name    TEXT NOT NULL,
    actor_profile TEXT NOT NULL,
    reason        TEXT NOT NULL DEFAULT '',
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX task_status_changes_entity_idx ON task_status_changes (task_id, changed_at);

CREATE TABLE bug_status_changes (
    id            UUID PRIMARY KEY,
    bug_id        UUID NOT NULL,
    from_status   TEXT NOT NULL,
    to_status     TEXT NOT NULL,
    actor_name    TEXT NOT NULL,
    actor_profile TEXT NOT NULL,
    reason        TEXT NOT NULL DEFAULT '',
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX bug_status_changes_entity_idx ON bug_status_changes (bug_id, changed_at);

CREATE TRIGGER project_status_changes_immutable
    BEFORE UPDATE OR DELETE ON project_status_changes
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();
CREATE TRIGGER project_status_changes_no_truncate
    BEFORE TRUNCATE ON project_status_changes
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER stage_status_changes_immutable
    BEFORE UPDATE OR DELETE ON stage_status_changes
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();
CREATE TRIGGER stage_status_changes_no_truncate
    BEFORE TRUNCATE ON stage_status_changes
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER task_status_changes_immutable
    BEFORE UPDATE OR DELETE ON task_status_changes
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();
CREATE TRIGGER task_status_changes_no_truncate
    BEFORE TRUNCATE ON task_status_changes
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER bug_status_changes_immutable
    BEFORE UPDATE OR DELETE ON bug_status_changes
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();
CREATE TRIGGER bug_status_changes_no_truncate
    BEFORE TRUNCATE ON bug_status_changes
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

-- +goose Down
DROP TABLE bug_status_changes;
DROP TABLE task_status_changes;
DROP TABLE stage_status_changes;
DROP TABLE project_status_changes;
