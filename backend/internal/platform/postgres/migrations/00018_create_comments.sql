-- +goose Up
-- Sin FK a sus elementos, igual que los hallazgos: un comentario no debe impedir
-- delete_task. La existencia se valida al crear.
CREATE TABLE comments (
    id            UUID PRIMARY KEY,
    item_kind     TEXT NOT NULL CHECK (item_kind IN ('project', 'stage', 'task', 'bug')),
    item_id       UUID NOT NULL,
    body          TEXT NOT NULL CHECK (body <> ''),
    actor_name    TEXT NOT NULL,
    actor_profile TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX comments_item_idx ON comments (item_kind, item_id, created_at);

CREATE TRIGGER comments_immutable
    BEFORE UPDATE OR DELETE ON comments
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER comments_no_truncate
    BEFORE TRUNCATE ON comments
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

-- +goose Down
DROP TABLE comments;
