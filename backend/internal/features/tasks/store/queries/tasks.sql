-- name: CreateTask :one
INSERT INTO tasks (id, stage_id, title, description, status, blocked, version)
VALUES ($1, $2, $3, $4, $5, $6, $7)
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
SET title = @title, description = @description, version = version + 1, updated_at = now()
WHERE id = @id AND version = @expected_version
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks
WHERE id = $1;
