-- name: CreateStage :one
INSERT INTO stages (id, project_id, name, description, status, position)
SELECT @id::uuid, @project_id::uuid, @name::text, @description::text, @status::text, COALESCE(MAX(position), 0) + 1
FROM stages
WHERE project_id = @project_id::uuid
RETURNING *;

-- name: GetStage :one
SELECT * FROM stages
WHERE id = $1;

-- name: ListStages :many
SELECT * FROM stages
WHERE project_id = $1
ORDER BY position;

-- name: UpdateStage :one
UPDATE stages
SET name = $2, description = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateStageStatus :one
UPDATE stages
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteStage :execrows
DELETE FROM stages
WHERE id = $1;
