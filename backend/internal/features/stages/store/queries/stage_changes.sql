-- name: CreateStageChange :one
INSERT INTO stage_changes (id, stage_id, version, kind, name, description, reason, requested_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: MaxStageChangeVersion :one
SELECT COALESCE(MAX(version), 0)::integer FROM stage_changes
WHERE stage_id = $1;

-- name: HasStageChanges :one
SELECT EXISTS (SELECT 1 FROM stage_changes WHERE stage_id = $1);

-- name: HasStageAdd :one
SELECT EXISTS (SELECT 1 FROM stage_changes WHERE stage_id = $1 AND kind = 'add');

-- name: ListStageChanges :many
SELECT sqlc.embed(c), e.name AS original_name, e.description AS original_description, e.created_at AS original_created_at
FROM stage_changes c
JOIN stages e ON e.id = c.stage_id
WHERE e.project_id = $1
ORDER BY e.created_at, e.id, c.version;

-- name: ListStageChangesByStage :many
SELECT sqlc.embed(c), e.name AS original_name, e.description AS original_description, e.created_at AS original_created_at
FROM stage_changes c
JOIN stages e ON e.id = c.stage_id
WHERE e.project_id = $1 AND c.stage_id = $2
ORDER BY c.version;
