-- name: CreateTaskStatusChange :exec
INSERT INTO task_status_changes (id, task_id, from_status, to_status, actor_name, actor_profile, reason)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListTaskStatusChanges :many
SELECT * FROM task_status_changes
WHERE task_id = $1
ORDER BY changed_at, id;
