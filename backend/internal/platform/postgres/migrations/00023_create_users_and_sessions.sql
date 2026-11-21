-- +goose Up
-- username es el nombre del actor en PMT: queda en historiales, comentarios y
-- hallazgos, por eso no cambia. No coincide con el nombre de ningún agente; eso
-- lo valida la aplicación.
CREATE TABLE users (
    id                UUID PRIMARY KEY,
    email             TEXT NOT NULL UNIQUE CHECK (email <> '' AND email = lower(email)),
    username          TEXT NOT NULL UNIQUE CHECK (username ~ '^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$'),
    name              TEXT NOT NULL CHECK (name <> ''),
    password_hash     TEXT NOT NULL,
    is_instance_owner BOOLEAN NOT NULL DEFAULT false,
    disabled_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id         BYTEA PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id_idx ON sessions(user_id);

-- +goose Down
DROP TABLE sessions;
DROP TABLE users;
