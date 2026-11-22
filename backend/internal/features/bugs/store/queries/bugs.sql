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
-- Guarda el status y quién tiene tomado el bug. taken_at solo cambia cuando lo
-- toma otra persona.
UPDATE bugs
SET status = @status::text,
    taken_by = sqlc.narg(taken_by)::uuid,
    taken_by_name = sqlc.narg(taken_by_name)::text,
    taken_at = CASE
        WHEN sqlc.narg(taken_by)::uuid IS NULL THEN NULL
        WHEN bugs.taken_by IS NOT DISTINCT FROM sqlc.narg(taken_by)::uuid THEN bugs.taken_at
        ELSE clock_timestamp()
    END,
    updated_at = clock_timestamp()
WHERE id = @id::uuid
RETURNING *;

-- name: ReleaseBug :one
-- Deja sin tomar un bug que sigue en manos de quien se esperaba.
UPDATE bugs
SET taken_by = NULL, taken_by_name = NULL, taken_at = NULL, updated_at = clock_timestamp()
WHERE id = @id::uuid AND taken_by = @taken_by::uuid
RETURNING *;

-- name: AdoptBugHolds :execrows
-- Deja tomados por el usuario los bugs en corrección que nadie tiene: los que
-- existían cuando la migración de bloqueos no encontró usuarios.
UPDATE bugs
SET taken_by = @user_id::uuid, taken_by_name = @username::text, taken_at = clock_timestamp()
WHERE taken_by IS NULL AND status IN ('in_dev', 'in_review', 'changes_requested', 'in_qa', 'pending_close');

-- name: LockBug :one
SELECT * FROM bugs
WHERE id = $1
FOR UPDATE;
