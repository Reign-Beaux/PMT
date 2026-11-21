-- name: HasUsers :one
SELECT EXISTS (SELECT 1 FROM users);

-- name: CreateOwnerIfNoUsers :execrows
INSERT INTO users (id, email, username, name, password_hash, is_instance_owner)
SELECT $1, $2, $3, $4, $5, true
WHERE NOT EXISTS (SELECT 1 FROM users);

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY username;

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetUserBySession :one
SELECT u.* FROM users u
JOIN sessions s ON s.user_id = u.id
WHERE s.id = $1 AND s.expires_at > now() AND u.disabled_at IS NULL;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions
WHERE user_id = $1;

-- name: DeleteOtherSessions :exec
DELETE FROM sessions
WHERE user_id = $1 AND id <> $2;

-- name: SetPasswordHash :exec
UPDATE users
SET password_hash = $2, updated_at = now()
WHERE id = $1;

-- name: DisableUser :one
UPDATE users
SET disabled_at = COALESCE(disabled_at, now()), updated_at = now()
WHERE id = $1
RETURNING *;

-- name: EnableUser :one
UPDATE users
SET disabled_at = NULL, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateAPIToken :one
INSERT INTO api_tokens (id, user_id, name, token_hash)
VALUES ($1, $2, $3, $4)
RETURNING created_at;

-- name: ListAPITokens :many
SELECT id, name, created_at, last_used_at FROM api_tokens
WHERE user_id = $1 AND revoked_at IS NULL
ORDER BY created_at DESC, id;

-- name: RevokeAPIToken :execrows
UPDATE api_tokens
SET revoked_at = now()
WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL;

-- name: RevokeUserAPITokens :exec
UPDATE api_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: TouchAPIToken :one
UPDATE api_tokens t
SET last_used_at = now()
FROM users u
WHERE t.token_hash = $1 AND t.revoked_at IS NULL
  AND u.id = t.user_id AND u.disabled_at IS NULL
RETURNING u.*;

-- name: CreateInvitation :one
INSERT INTO invitations (id, email, token_hash, invited_by, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING expires_at;

-- name: ListInvitations :many
SELECT id, email, expires_at, accepted_at FROM invitations
ORDER BY created_at DESC, id;

-- name: GetInvitationByHash :one
SELECT email, expires_at, accepted_at,
       EXISTS (SELECT 1 FROM users WHERE users.email = invitations.email) AS user_exists
FROM invitations
WHERE token_hash = $1;

-- name: LockInvitationByHash :one
SELECT id, email, expires_at, accepted_at FROM invitations
WHERE token_hash = $1
FOR UPDATE;

-- name: MarkInvitationAccepted :exec
UPDATE invitations
SET accepted_at = now()
WHERE id = $1;

-- name: CreateInvitedUser :one
INSERT INTO users (id, email, username, name, password_hash, is_instance_owner)
VALUES ($1, $2, $3, $4, $5, false)
RETURNING *;
