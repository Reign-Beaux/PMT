-- Remove orphaned projects that predate user ownership.
TRUNCATE projects CASCADE;

ALTER TABLE projects
    ADD COLUMN user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;
