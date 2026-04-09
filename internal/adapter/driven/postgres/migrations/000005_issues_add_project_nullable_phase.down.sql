DROP INDEX IF EXISTS idx_issues_project_id;
DROP INDEX IF EXISTS idx_issues_phase_id;

ALTER TABLE issues ALTER COLUMN phase_id SET NOT NULL;
ALTER TABLE issues DROP COLUMN project_id;

CREATE INDEX idx_issues_phase_id ON issues(phase_id);
