CREATE TABLE labels (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID         NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    color      VARCHAR(7)   NOT NULL DEFAULT '#6366f1',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, name)
);

CREATE TABLE issue_labels (
    issue_id  UUID NOT NULL REFERENCES issues(id)  ON DELETE CASCADE,
    label_id  UUID NOT NULL REFERENCES labels(id)  ON DELETE CASCADE,
    PRIMARY KEY (issue_id, label_id)
);

CREATE INDEX idx_labels_project_id    ON labels(project_id);
CREATE INDEX idx_issue_labels_issue   ON issue_labels(issue_id);
CREATE INDEX idx_issue_labels_label   ON issue_labels(label_id);
