-- +goose Up
-- stage_id y task_id van sin FK, igual que los hallazgos: un plan no debe impedir
-- delete_stage ni delete_task. La existencia se valida al crear y al revisar.
CREATE TABLE test_plans (
    id           UUID PRIMARY KEY,
    stage_id     UUID NOT NULL,
    title        TEXT NOT NULL CHECK (title <> ''),
    status       TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'approved', 'returned')),
    created_by   TEXT NOT NULL,
    review_notes TEXT NOT NULL DEFAULT '',
    reviewed_by  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX test_plans_stage_idx ON test_plans (stage_id);

CREATE TABLE test_plan_versions (
    plan_id     UUID NOT NULL REFERENCES test_plans (id) ON DELETE RESTRICT,
    version     INTEGER NOT NULL CHECK (version > 0),
    happy_path  TEXT NOT NULL,
    error_cases TEXT NOT NULL,
    reason      TEXT NOT NULL DEFAULT '',
    created_by  TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (plan_id, version)
);

CREATE TRIGGER test_plan_versions_immutable
    BEFORE UPDATE OR DELETE ON test_plan_versions
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER test_plan_versions_no_truncate
    BEFORE TRUNCATE ON test_plan_versions
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

CREATE TABLE test_plan_tasks (
    plan_id UUID NOT NULL REFERENCES test_plans (id) ON DELETE CASCADE,
    task_id UUID NOT NULL,
    PRIMARY KEY (plan_id, task_id)
);

CREATE INDEX test_plan_tasks_task_idx ON test_plan_tasks (task_id);

-- +goose Down
DROP TABLE test_plan_tasks;
DROP TABLE test_plan_versions;
DROP TABLE test_plans;
