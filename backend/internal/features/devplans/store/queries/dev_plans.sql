-- name: CreateDevPlan :exec
INSERT INTO dev_plans (id, stage_id, bug_id, created_by)
VALUES ($1, $2, $3, $4);

-- name: InsertDevPlanVersion :exec
INSERT INTO dev_plan_versions (plan_id, version, scope, technical_decisions, screen_states, edge_cases, error_handling, acceptance_criteria, reason, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: LinkDevPlanTasks :exec
INSERT INTO dev_plan_tasks (plan_id, task_id)
SELECT $1, unnest(@task_ids::uuid[]);

-- name: UnlinkDevPlanTasks :exec
DELETE FROM dev_plan_tasks WHERE plan_id = $1;

-- name: LockDevPlan :one
SELECT id FROM dev_plans WHERE id = $1 FOR UPDATE;

-- name: TouchDevPlan :exec
UPDATE dev_plans SET updated_at = clock_timestamp() WHERE id = $1;

-- name: GetDevPlan :one
SELECT p.id, p.stage_id, p.bug_id, p.created_by, p.created_at, p.updated_at,
       v.version, v.scope, v.technical_decisions, v.screen_states, v.edge_cases,
       v.error_handling, v.acceptance_criteria, v.reason,
       v.created_by AS version_created_by, v.created_at AS version_created_at,
       ARRAY(SELECT t.task_id FROM dev_plan_tasks t WHERE t.plan_id = p.id ORDER BY t.task_id)::uuid[] AS task_ids
FROM dev_plans p
JOIN LATERAL (
    SELECT * FROM dev_plan_versions dv WHERE dv.plan_id = p.id ORDER BY dv.version DESC LIMIT 1
) v ON TRUE
WHERE p.id = $1;

-- name: ListDevPlans :many
SELECT p.id, p.stage_id, p.bug_id, p.created_by, p.created_at, p.updated_at,
       v.version, v.scope, v.technical_decisions, v.screen_states, v.edge_cases,
       v.error_handling, v.acceptance_criteria, v.reason,
       v.created_by AS version_created_by, v.created_at AS version_created_at,
       ARRAY(SELECT t.task_id FROM dev_plan_tasks t WHERE t.plan_id = p.id ORDER BY t.task_id)::uuid[] AS task_ids
FROM dev_plans p
JOIN LATERAL (
    SELECT * FROM dev_plan_versions dv WHERE dv.plan_id = p.id ORDER BY dv.version DESC LIMIT 1
) v ON TRUE
WHERE (sqlc.narg(stage_id)::uuid IS NULL OR p.stage_id = sqlc.narg(stage_id)::uuid)
  AND (sqlc.narg(bug_id)::uuid IS NULL OR p.bug_id = sqlc.narg(bug_id)::uuid)
  AND (sqlc.narg(task_id)::uuid IS NULL OR EXISTS (
      SELECT 1 FROM dev_plan_tasks t WHERE t.plan_id = p.id AND t.task_id = sqlc.narg(task_id)::uuid))
ORDER BY p.created_at, p.id;

-- name: ListDevPlanVersions :many
SELECT * FROM dev_plan_versions WHERE plan_id = $1 ORDER BY version;
