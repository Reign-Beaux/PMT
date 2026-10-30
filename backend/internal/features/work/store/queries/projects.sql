-- name: CreateProject :one
INSERT INTO projects (id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects
WHERE id = $1;

-- name: ListProjects :many
SELECT * FROM projects
ORDER BY created_at DESC, id;

-- name: UpdateProject :one
UPDATE projects
SET name = $2, description = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteProject :execrows
DELETE FROM projects
WHERE id = $1;
