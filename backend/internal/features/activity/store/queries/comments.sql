-- name: CreateComment :one
INSERT INTO comments (id, item_kind, item_id, body, actor_name, actor_profile)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListComments :many
SELECT * FROM comments
WHERE item_kind = $1 AND item_id = $2
ORDER BY created_at, id;
