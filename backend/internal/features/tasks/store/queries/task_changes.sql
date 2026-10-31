-- name: CreateTaskChange :one
INSERT INTO task_changes (id, task_id, version, kind, title, description, reason, requested_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: MaxTaskChangeVersion :one
SELECT COALESCE(MAX(version), 0)::integer FROM task_changes
WHERE task_id = $1;

-- name: HasTaskChanges :one
SELECT EXISTS (SELECT 1 FROM task_changes WHERE task_id = $1);

-- name: ListTaskChanges :many
SELECT sqlc.embed(c), e.title AS original_title, e.description AS original_description, e.created_at AS original_created_at
FROM task_changes c
JOIN tasks e ON e.id = c.task_id
WHERE e.stage_id = $1
ORDER BY e.created_at, e.id, c.version;

-- name: ListTaskChangesByTask :many
SELECT sqlc.embed(c), e.title AS original_title, e.description AS original_description, e.created_at AS original_created_at
FROM task_changes c
JOIN tasks e ON e.id = c.task_id
WHERE e.stage_id = $1 AND c.task_id = $2
ORDER BY c.version;
