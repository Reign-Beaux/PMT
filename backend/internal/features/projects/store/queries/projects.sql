-- name: CreateProject :one
INSERT INTO projects (id, name, description, status, product_version)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects
WHERE id = $1;

-- name: ListProjects :many
SELECT * FROM projects
ORDER BY created_at DESC, id;

-- name: UpdateProject :one
UPDATE projects
SET name = $2, description = $3, product_version = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateProjectStatus :one
UPDATE projects
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteProject :execrows
DELETE FROM projects
WHERE id = $1;

-- name: LockProject :one
SELECT * FROM projects
WHERE id = $1
FOR UPDATE;

-- name: UpdateProjectProductVersion :one
UPDATE projects
SET product_version = $2, updated_at = now()
WHERE id = $1
RETURNING *;
