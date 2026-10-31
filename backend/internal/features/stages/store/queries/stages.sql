-- name: CreateStage :one
INSERT INTO stages (id, project_id, name, description, status, position, add_reason, add_requested_by)
SELECT @id::uuid, @project_id::uuid, @name::text, @description::text, @status::text, COALESCE(MAX(position), 0) + 1, NULLIF(@add_reason::text, ''), NULLIF(@add_requested_by::text, '')
FROM stages
WHERE project_id = @project_id::uuid
RETURNING *;

-- name: GetStage :one
SELECT s.id, s.project_id, COALESCE(c.name, s.name)::text AS name, COALESCE(c.description, s.description)::text AS description, s.position, s.created_at, s.updated_at, s.status, s.retired_at, s.add_reason, s.add_requested_by, s.lead
FROM stages s
LEFT JOIN LATERAL (
    SELECT name, description FROM stage_changes
    WHERE stage_id = s.id
    ORDER BY version DESC
    LIMIT 1
) c ON true
WHERE s.id = $1 AND s.retired_at IS NULL;

-- name: LockStage :one
SELECT * FROM stages
WHERE id = $1 AND retired_at IS NULL
FOR UPDATE;

-- name: ListStages :many
SELECT s.id, s.project_id, COALESCE(c.name, s.name)::text AS name, COALESCE(c.description, s.description)::text AS description, s.position, s.created_at, s.updated_at, s.status, s.retired_at, s.add_reason, s.add_requested_by, s.lead
FROM stages s
LEFT JOIN LATERAL (
    SELECT name, description FROM stage_changes
    WHERE stage_id = s.id
    ORDER BY version DESC
    LIMIT 1
) c ON true
WHERE s.project_id = $1 AND s.retired_at IS NULL
ORDER BY s.position;

-- name: UpdateStage :one
UPDATE stages
SET name = $2, description = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateStageStatus :one
WITH updated AS (
    UPDATE stages
    SET status = $2, updated_at = now()
    WHERE stages.id = $1
    RETURNING *
)
SELECT s.id, s.project_id, COALESCE(c.name, s.name)::text AS name, COALESCE(c.description, s.description)::text AS description, s.position, s.created_at, s.updated_at, s.status, s.retired_at, s.add_reason, s.add_requested_by, s.lead
FROM updated s
LEFT JOIN LATERAL (
    SELECT name, description FROM stage_changes
    WHERE stage_id = s.id
    ORDER BY version DESC
    LIMIT 1
) c ON true;

-- name: DeleteStage :execrows
DELETE FROM stages
WHERE id = $1;

-- name: RetireStage :exec
UPDATE stages
SET retired_at = now(), updated_at = now()
WHERE id = $1;

-- name: LeaveStagePlanning :exec
UPDATE stages
SET status = $2, add_reason = NULL, add_requested_by = NULL, updated_at = now()
WHERE id = $1;

-- name: UpdateStageLead :execrows
UPDATE stages
SET lead = $2, updated_at = now()
WHERE id = $1 AND retired_at IS NULL;

-- name: AddStageMember :exec
INSERT INTO stage_members (stage_id, agent, added_by)
VALUES ($1, $2, $3);

-- name: RemoveStageMember :execrows
DELETE FROM stage_members
WHERE stage_id = $1 AND agent = $2;

-- name: ListStageMembers :many
SELECT agent FROM stage_members
WHERE stage_id = $1
ORDER BY added_at, agent;

-- name: ListProjectStageMembers :many
SELECT m.stage_id, m.agent
FROM stage_members m
JOIN stages s ON s.id = m.stage_id
WHERE s.project_id = $1
ORDER BY m.added_at, m.agent;
