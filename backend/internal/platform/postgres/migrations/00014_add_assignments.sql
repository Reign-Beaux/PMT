-- +goose Up
-- Los agentes se referencian por nombre, que es único y no cambia: así las salidas
-- muestran el nombre sin que otra feature lea agents.
ALTER TABLE stages ADD COLUMN lead TEXT REFERENCES agents (name) ON DELETE RESTRICT;
ALTER TABLE tasks ADD COLUMN assignee TEXT REFERENCES agents (name) ON DELETE RESTRICT;

-- CASCADE porque el equipo no debe impedir delete_stage en planning.
CREATE TABLE stage_members (
    stage_id UUID NOT NULL REFERENCES stages (id) ON DELETE CASCADE,
    agent    TEXT NOT NULL REFERENCES agents (name) ON DELETE RESTRICT,
    added_by TEXT NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (stage_id, agent)
);

CREATE INDEX stage_members_agent_idx ON stage_members (agent);

-- +goose Down
DROP TABLE stage_members;
ALTER TABLE tasks DROP COLUMN assignee;
ALTER TABLE stages DROP COLUMN lead;
