-- +goose Up
CREATE TABLE agents (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE CHECK (name <> ''),
    profile    TEXT NOT NULL CHECK (profile IN (
        'PP', 'PC', 'TL', 'DevFrontend', 'DevBackend', 'CR', 'QAP', 'QA', 'User'
    )),
    token_hash BYTEA NOT NULL UNIQUE,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

-- +goose Down
DROP TABLE agents;
