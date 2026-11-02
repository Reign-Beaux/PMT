-- name: CreateBug :one
INSERT INTO bugs (id, project_id, title, description, steps_to_reproduce, product_version, status, reported_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetBug :one
SELECT * FROM bugs
WHERE id = $1;

-- name: ListBugs :many
SELECT * FROM bugs
WHERE project_id = $1
ORDER BY created_at, id;

-- name: UpdateBug :one
UPDATE bugs
SET title = $2, description = $3, steps_to_reproduce = $4, product_version = $5, updated_at = clock_timestamp()
WHERE id = $1
RETURNING *;

-- name: UpdateBugStatus :one
UPDATE bugs
SET status = $2, updated_at = clock_timestamp()
WHERE id = $1
RETURNING *;

-- name: LockBug :one
SELECT * FROM bugs
WHERE id = $1
FOR UPDATE;
