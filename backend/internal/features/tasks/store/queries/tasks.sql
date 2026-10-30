-- name: CreateTask :one
INSERT INTO tasks (id, stage_id, title, description, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = $1;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE stage_id = $1
ORDER BY created_at, id;

-- name: UpdateTask :one
UPDATE tasks
SET title = $2, description = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks
WHERE id = $1;
