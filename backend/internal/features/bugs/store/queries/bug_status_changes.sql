-- name: CreateBugStatusChange :exec
INSERT INTO bug_status_changes (id, bug_id, from_status, to_status, actor_name, actor_profile, reason)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListBugStatusChanges :many
SELECT * FROM bug_status_changes
WHERE bug_id = $1
ORDER BY changed_at, id;
