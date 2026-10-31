-- name: CreateAgent :one
INSERT INTO agents (id, name, profile, token_hash)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAgentByName :one
SELECT * FROM agents
WHERE name = $1;

-- name: GetActiveAgentByTokenHash :one
SELECT * FROM agents
WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: ListAgents :many
SELECT * FROM agents
ORDER BY name;

-- name: UpdateAgentTokenHash :execrows
UPDATE agents
SET token_hash = $2
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeAgent :execrows
UPDATE agents
SET revoked_at = clock_timestamp()
WHERE id = $1 AND revoked_at IS NULL;
