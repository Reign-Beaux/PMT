-- name: CreateFinding :one
INSERT INTO findings (id, item_kind, item_id, description, blocking, actor_name, actor_profile)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetFinding :one
SELECT * FROM findings WHERE id = $1;

-- name: CloseFinding :one
UPDATE findings
SET status = $2, closed_by = $3, close_reason = $4, closed_at = clock_timestamp()
WHERE id = $1 AND status = 'open'
RETURNING *;

-- name: ListFindings :many
SELECT * FROM findings
WHERE item_kind = $1 AND item_id = $2
ORDER BY created_at, id;

-- name: ListPendingFindings :many
SELECT * FROM findings
WHERE item_kind = $1 AND item_id = $2 AND status = 'open'
ORDER BY blocking DESC, created_at, id;
