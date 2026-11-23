-- +goose Up
-- Inicios de sesión del CLI pmt por navegador. El CLI pide un código, la persona
-- lo autoriza en /cli con su sesión y el CLI recibe un token de API al sondear.
-- Solo se guarda el SHA-256 del secreto de sondeo; el token se crea al
-- entregarlo, así que nunca queda en la base.
CREATE TABLE cli_logins (
    id          UUID PRIMARY KEY,
    code        TEXT NOT NULL UNIQUE,
    poll_hash   BYTEA NOT NULL UNIQUE,
    machine     TEXT NOT NULL CHECK (machine <> ''),
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'authorized', 'cancelled', 'consumed')),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    token_id    UUID REFERENCES api_tokens(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    decided_at  TIMESTAMPTZ,
    CHECK ((status IN ('authorized', 'consumed')) = (user_id IS NOT NULL))
);
CREATE INDEX cli_logins_expires_at_idx ON cli_logins(expires_at);

-- +goose Down
DROP TABLE cli_logins;
