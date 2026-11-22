-- +goose Up

-- Miembros de un proyecto: un solo owner, y editores y viewers. Borrar el
-- proyecto (solo en planning) borra sus miembros.
CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    added_at   TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (project_id, user_id)
);

CREATE UNIQUE INDEX project_members_one_owner_idx ON project_members (project_id) WHERE role = 'owner';
CREATE INDEX project_members_user_idx ON project_members (user_id);

-- Los proyectos que ya existen quedan del dueño de la instancia. Si todavía no
-- hay usuarios, los adopta al arrancar el servidor, después de crear al dueño.
INSERT INTO project_members (project_id, user_id, role)
SELECT p.id, u.id, 'owner'
FROM projects p
CROSS JOIN users u
WHERE u.is_instance_owner;

-- pmt_project_of da el proyecto de un elemento por su tipo e ID, o NULL si no
-- existe. Lo usan la comprobación de membresía y los avisos de cambios.
-- +goose StatementBegin
CREATE FUNCTION pmt_project_of(kind TEXT, item UUID) RETURNS UUID AS $$
BEGIN
    CASE kind
    WHEN 'project' THEN
        RETURN (SELECT id FROM projects WHERE id = item);
    WHEN 'stage' THEN
        RETURN (SELECT project_id FROM stages WHERE id = item);
    WHEN 'task' THEN
        RETURN (SELECT s.project_id FROM tasks t JOIN stages s ON s.id = t.stage_id WHERE t.id = item);
    WHEN 'bug' THEN
        RETURN (SELECT project_id FROM bugs WHERE id = item);
    WHEN 'finding' THEN
        RETURN (SELECT pmt_project_of(f.item_kind, f.item_id) FROM findings f WHERE f.id = item);
    WHEN 'test_plan' THEN
        RETURN (SELECT s.project_id FROM test_plans p JOIN stages s ON s.id = p.stage_id WHERE p.id = item);
    WHEN 'dev_plan' THEN
        RETURN (
            SELECT COALESCE(s.project_id, b.project_id)
            FROM dev_plans d
            LEFT JOIN stages s ON s.id = d.stage_id
            LEFT JOIN bugs b ON b.id = d.bug_id
            WHERE d.id = item
        );
    ELSE
        RETURN NULL;
    END CASE;
END;
$$ LANGUAGE plpgsql STABLE;
-- +goose StatementEnd

-- El aviso de cambios lleva el proyecto del registro, para que /api/events solo
-- lo entregue a quien lo ve. Un cambio de miembros lleva además el usuario
-- afectado, que lo recibe aunque ya no sea miembro.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION notify_pmt_change() RETURNS trigger AS $$
DECLARE
    rec     JSONB;
    project UUID;
    payload JSONB;
BEGIN
    IF TG_OP = 'DELETE' THEN
        rec := to_jsonb(OLD);
    ELSE
        rec := to_jsonb(NEW);
    END IF;

    IF TG_TABLE_NAME = 'projects' THEN
        project := (rec ->> 'id')::uuid;
    ELSIF rec ? 'project_id' THEN
        project := (rec ->> 'project_id')::uuid;
    ELSIF rec ? 'stage_id' THEN
        project := pmt_project_of('stage', (rec ->> 'stage_id')::uuid);
    ELSIF rec ? 'task_id' THEN
        project := pmt_project_of('task', (rec ->> 'task_id')::uuid);
    ELSIF rec ? 'bug_id' THEN
        project := pmt_project_of('bug', (rec ->> 'bug_id')::uuid);
    ELSIF rec ? 'item_kind' THEN
        project := pmt_project_of(rec ->> 'item_kind', (rec ->> 'item_id')::uuid);
    END IF;

    payload := jsonb_build_object('resource', TG_ARGV[0], 'project_id', project);
    IF TG_TABLE_NAME = 'project_members' THEN
        payload := payload || jsonb_build_object('user_id', rec ->> 'user_id');
    END IF;
    PERFORM pg_notify('pmt_changes', payload::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER project_members_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON project_members
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('project');

-- +goose Down
DROP TRIGGER project_members_notify_change ON project_members;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION notify_pmt_change() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('pmt_changes', json_build_object('resource', TG_ARGV[0])::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP FUNCTION pmt_project_of(TEXT, UUID);
DROP TABLE project_members;
