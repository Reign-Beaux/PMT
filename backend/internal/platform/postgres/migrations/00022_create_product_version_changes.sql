-- +goose Up
-- Historial de la versión de producto de un proyecto en working. Como el
-- historial de status, no tiene llave foránea: no impide eliminar el proyecto si
-- vuelve a planning y sobrevive a él, consultable por su ID.
CREATE TABLE product_version_changes (
    id            UUID PRIMARY KEY,
    project_id    UUID NOT NULL,
    from_version  TEXT NOT NULL,
    to_version    TEXT NOT NULL CHECK (to_version <> ''),
    actor_name    TEXT NOT NULL,
    actor_profile TEXT NOT NULL,
    reason        TEXT NOT NULL CHECK (reason <> ''),
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX product_version_changes_entity_idx ON product_version_changes (project_id, changed_at);

CREATE TRIGGER product_version_changes_immutable
    BEFORE UPDATE OR DELETE ON product_version_changes
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();
CREATE TRIGGER product_version_changes_no_truncate
    BEFORE TRUNCATE ON product_version_changes
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER product_version_changes_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON product_version_changes
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('project');

-- +goose Down
DROP TABLE product_version_changes;
