-- +goose Up
-- Sin FK a tasks ni bugs, igual que el historial de status: un hallazgo no debe
-- impedir delete_task. La existencia se valida al crear.
CREATE TABLE findings (
    id            UUID PRIMARY KEY,
    item_kind     TEXT NOT NULL CHECK (item_kind IN ('task', 'bug')),
    item_id       UUID NOT NULL,
    description   TEXT NOT NULL CHECK (description <> ''),
    blocking      BOOLEAN NOT NULL,
    status        TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'fixed', 'dismissed')),
    actor_name    TEXT NOT NULL,
    actor_profile TEXT NOT NULL,
    closed_by     TEXT,
    close_reason  TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    closed_at     TIMESTAMPTZ
);

CREATE INDEX findings_item_idx ON findings (item_kind, item_id, status);

-- +goose Down
DROP TABLE findings;
