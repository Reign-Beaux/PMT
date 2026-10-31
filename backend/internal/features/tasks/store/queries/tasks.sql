-- name: CreateTask :one
INSERT INTO tasks (id, stage_id, title, description, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetTask :one
SELECT s.id, s.stage_id, COALESCE(c.title, s.title)::text AS title, COALESCE(c.description, s.description)::text AS description, s.status, s.created_at, s.updated_at, s.retired_at, s.assignee
FROM tasks s
LEFT JOIN LATERAL (
    SELECT title, description FROM task_changes
    WHERE task_id = s.id
    ORDER BY version DESC
    LIMIT 1
) c ON true
WHERE s.id = $1 AND s.retired_at IS NULL;

-- name: LockTask :one
SELECT * FROM tasks
WHERE id = $1 AND retired_at IS NULL
FOR UPDATE;

-- name: ListTasks :many
SELECT s.id, s.stage_id, COALESCE(c.title, s.title)::text AS title, COALESCE(c.description, s.description)::text AS description, s.status, s.created_at, s.updated_at, s.retired_at, s.assignee
FROM tasks s
LEFT JOIN LATERAL (
    SELECT title, description FROM task_changes
    WHERE task_id = s.id
    ORDER BY version DESC
    LIMIT 1
) c ON true
WHERE s.stage_id = @stage_id AND s.retired_at IS NULL AND (@status::text = '' OR s.status = @status::text)
ORDER BY s.created_at, s.id;

-- name: UpdateTask :one
UPDATE tasks
SET title = $2, description = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateTaskStatus :one
WITH updated AS (
    UPDATE tasks
    SET status = @status, assignee = NULLIF(@assignee::text, ''), updated_at = now()
    WHERE tasks.id = @id AND tasks.retired_at IS NULL
    RETURNING *
)
SELECT s.id, s.stage_id, COALESCE(c.title, s.title)::text AS title, COALESCE(c.description, s.description)::text AS description, s.status, s.created_at, s.updated_at, s.retired_at, s.assignee
FROM updated s
LEFT JOIN LATERAL (
    SELECT title, description FROM task_changes
    WHERE task_id = s.id
    ORDER BY version DESC
    LIMIT 1
) c ON true;

-- name: DeleteTask :execrows
DELETE FROM tasks
WHERE id = $1;

-- name: RetireTask :exec
UPDATE tasks
SET retired_at = now(), updated_at = now()
WHERE id = $1;

-- name: UpdateTaskAssignee :one
WITH updated AS (
    UPDATE tasks
    SET assignee = $2, updated_at = now()
    WHERE tasks.id = $1 AND tasks.retired_at IS NULL
    RETURNING *
)
SELECT s.id, s.stage_id, COALESCE(c.title, s.title)::text AS title, COALESCE(c.description, s.description)::text AS description, s.status, s.created_at, s.updated_at, s.retired_at, s.assignee
FROM updated s
LEFT JOIN LATERAL (
    SELECT title, description FROM task_changes
    WHERE task_id = s.id
    ORDER BY version DESC
    LIMIT 1
) c ON true;

-- name: HasOpenAssignedTasks :one
SELECT EXISTS (
    SELECT 1 FROM tasks
    WHERE stage_id = $1 AND assignee = $2 AND retired_at IS NULL
      AND status NOT IN ('done', 'cancelled')
);
