-- name: CreateProject :one
-- Crea el proyecto y a su owner en una sola sentencia.
WITH created AS (
    INSERT INTO projects (id, name, description, status, product_version)
    VALUES (@id, @name, @description, @status, @product_version)
    RETURNING *
), owner AS (
    INSERT INTO project_members (project_id, user_id, role)
    SELECT created.id, @owner_id, 'owner' FROM created
)
SELECT * FROM created;

-- name: GetProject :one
SELECT * FROM projects
WHERE id = $1;

-- name: ListProjects :many
-- Los proyectos del usuario en los que tiene uno de los roles dados.
SELECT p.* FROM projects p
JOIN project_members m ON m.project_id = p.id AND m.user_id = @user_id
WHERE m.role = ANY(@roles::text[])
ORDER BY p.created_at DESC, p.id;

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
