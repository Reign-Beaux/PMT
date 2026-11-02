-- name: CreateStageStatusChange :exec
INSERT INTO stage_status_changes (id, stage_id, from_status, to_status, actor_name, actor_profile, reason)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListStageStatusChanges :many
SELECT * FROM stage_status_changes
WHERE stage_id = $1
ORDER BY changed_at, id;
