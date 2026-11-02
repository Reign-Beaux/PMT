-- +goose Up
ALTER TABLE agents DROP CONSTRAINT agents_profile_check;
ALTER TABLE agents ADD CONSTRAINT agents_profile_check CHECK (profile IN (
    'PP', 'PC', 'TL', 'DevFrontend', 'DevBackend', 'CR', 'QAP', 'QA', 'DP', 'User'
));

-- stage_id, bug_id y task_id van sin FK, igual que los planes QAP: un plan no debe
-- impedir delete_stage ni delete_task. La existencia se valida al crear y al revisar.
CREATE TABLE dev_plans (
    id         UUID PRIMARY KEY,
    stage_id   UUID,
    bug_id     UUID,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CHECK ((stage_id IS NULL) <> (bug_id IS NULL))
);

CREATE INDEX dev_plans_stage_idx ON dev_plans (stage_id);
CREATE INDEX dev_plans_bug_idx ON dev_plans (bug_id);

CREATE TABLE dev_plan_versions (
    plan_id             UUID NOT NULL REFERENCES dev_plans (id) ON DELETE RESTRICT,
    version             INTEGER NOT NULL CHECK (version > 0),
    scope               TEXT NOT NULL CHECK (scope <> ''),
    technical_decisions TEXT NOT NULL CHECK (technical_decisions <> ''),
    screen_states       TEXT NOT NULL CHECK (screen_states <> ''),
    edge_cases          TEXT NOT NULL CHECK (edge_cases <> ''),
    error_handling      TEXT NOT NULL CHECK (error_handling <> ''),
    acceptance_criteria TEXT NOT NULL CHECK (acceptance_criteria <> ''),
    reason              TEXT NOT NULL DEFAULT '',
    created_by          TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (plan_id, version)
);

CREATE TRIGGER dev_plan_versions_immutable
    BEFORE UPDATE OR DELETE ON dev_plan_versions
    FOR EACH ROW EXECUTE FUNCTION reject_change_mutation();

CREATE TRIGGER dev_plan_versions_no_truncate
    BEFORE TRUNCATE ON dev_plan_versions
    FOR EACH STATEMENT EXECUTE FUNCTION reject_change_mutation();

CREATE TABLE dev_plan_tasks (
    plan_id UUID NOT NULL REFERENCES dev_plans (id) ON DELETE CASCADE,
    task_id UUID NOT NULL,
    PRIMARY KEY (plan_id, task_id)
);

CREATE INDEX dev_plan_tasks_task_idx ON dev_plan_tasks (task_id);

-- +goose Down
DROP TABLE dev_plan_tasks;
DROP TABLE dev_plan_versions;
DROP TABLE dev_plans;

ALTER TABLE agents DROP CONSTRAINT agents_profile_check;
ALTER TABLE agents ADD CONSTRAINT agents_profile_check CHECK (profile IN (
    'PP', 'PC', 'TL', 'DevFrontend', 'DevBackend', 'CR', 'QAP', 'QA', 'User'
));
