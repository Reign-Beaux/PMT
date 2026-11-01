-- name: CreateTestPlan :exec
INSERT INTO test_plans (id, stage_id, title, created_by)
VALUES ($1, $2, $3, $4);

-- name: InsertTestPlanVersion :exec
INSERT INTO test_plan_versions (plan_id, version, happy_path, error_cases, reason, created_by)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: LinkTestPlanTasks :exec
INSERT INTO test_plan_tasks (plan_id, task_id)
SELECT $1, unnest(@task_ids::uuid[]);

-- name: UnlinkTestPlanTasks :exec
DELETE FROM test_plan_tasks WHERE plan_id = $1;

-- name: LockTestPlan :one
SELECT id FROM test_plans WHERE id = $1 FOR UPDATE;

-- name: UpdateTestPlanReview :exec
UPDATE test_plans
SET status = $2, review_notes = $3, reviewed_by = $4, updated_at = clock_timestamp()
WHERE id = $1;

-- name: GetTestPlan :one
SELECT p.id, p.stage_id, p.title, p.status, p.created_by, p.review_notes, p.reviewed_by, p.created_at, p.updated_at,
       v.version, v.happy_path, v.error_cases, v.reason,
       v.created_by AS version_created_by, v.created_at AS version_created_at,
       ARRAY(SELECT t.task_id FROM test_plan_tasks t WHERE t.plan_id = p.id ORDER BY t.task_id)::uuid[] AS task_ids
FROM test_plans p
JOIN LATERAL (
    SELECT * FROM test_plan_versions tv WHERE tv.plan_id = p.id ORDER BY tv.version DESC LIMIT 1
) v ON TRUE
WHERE p.id = $1;

-- name: ListTestPlans :many
SELECT p.id, p.stage_id, p.title, p.status, p.created_by, p.review_notes, p.reviewed_by, p.created_at, p.updated_at,
       v.version, v.happy_path, v.error_cases, v.reason,
       v.created_by AS version_created_by, v.created_at AS version_created_at,
       ARRAY(SELECT t.task_id FROM test_plan_tasks t WHERE t.plan_id = p.id ORDER BY t.task_id)::uuid[] AS task_ids
FROM test_plans p
JOIN LATERAL (
    SELECT * FROM test_plan_versions tv WHERE tv.plan_id = p.id ORDER BY tv.version DESC LIMIT 1
) v ON TRUE
WHERE p.stage_id = @stage_id
  AND (sqlc.narg(task_id)::uuid IS NULL OR EXISTS (
      SELECT 1 FROM test_plan_tasks t WHERE t.plan_id = p.id AND t.task_id = sqlc.narg(task_id)::uuid))
ORDER BY p.created_at, p.id;

-- name: ListTestPlanVersions :many
SELECT * FROM test_plan_versions WHERE plan_id = $1 ORDER BY version;
