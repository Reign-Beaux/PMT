-- +goose Up
CREATE TABLE invitations (
    id          UUID PRIMARY KEY,
    email       TEXT NOT NULL CHECK (email <> '' AND email = lower(email)),
    token_hash  BYTEA NOT NULL UNIQUE,
    invited_by  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE invitations;
