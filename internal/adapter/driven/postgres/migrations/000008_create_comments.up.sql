CREATE TABLE comments (
    id         UUID  PRIMARY KEY DEFAULT uuid_generate_v4(),
    issue_id   UUID  NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    body       TEXT  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_issue_id ON comments(issue_id);
