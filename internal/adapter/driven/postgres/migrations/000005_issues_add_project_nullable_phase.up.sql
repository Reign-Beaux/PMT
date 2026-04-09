ALTER TABLE issues ADD COLUMN project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE;
ALTER TABLE issues ALTER COLUMN phase_id DROP NOT NULL;

DROP INDEX IF EXISTS idx_issues_phase_id;
CREATE INDEX idx_issues_phase_id  ON issues(phase_id) WHERE phase_id IS NOT NULL;
CREATE INDEX idx_issues_project_id ON issues(project_id);
