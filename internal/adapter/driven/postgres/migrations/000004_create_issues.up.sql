CREATE TABLE issues (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    phase_id   UUID         NOT NULL REFERENCES phases(id) ON DELETE CASCADE,
    title      VARCHAR(500) NOT NULL,
    spec       TEXT         NOT NULL DEFAULT '',
    status     VARCHAR(50)  NOT NULL DEFAULT 'open',
    priority   VARCHAR(50)  NOT NULL DEFAULT 'medium',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_issues_phase_id ON issues(phase_id);
