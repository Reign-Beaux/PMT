ALTER TABLE issues ADD COLUMN project_id UUID REFERENCES projects(id) ON DELETE CASCADE;

UPDATE issues i
SET project_id = p.project_id
FROM phases p
WHERE i.phase_id = p.id;

DELETE FROM issues WHERE project_id IS NULL;

ALTER TABLE issues ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE issues ALTER COLUMN phase_id DROP NOT NULL;

DROP INDEX IF EXISTS idx_issues_phase_id;
CREATE INDEX idx_issues_phase_id  ON issues(phase_id) WHERE phase_id IS NOT NULL;
CREATE INDEX idx_issues_project_id ON issues(project_id);
