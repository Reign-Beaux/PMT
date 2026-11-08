-- +goose Up

-- +goose StatementBegin
CREATE FUNCTION notify_pmt_change() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('pmt_changes', json_build_object('resource', TG_ARGV[0])::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- FOR EACH ROW: un UPDATE o DELETE sin filas no publica. PostgreSQL entrega el
-- NOTIFY solo al commit y junta payloads idénticos de una transacción.
CREATE TRIGGER projects_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON projects
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('project');

CREATE TRIGGER stages_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON stages
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('stage');

CREATE TRIGGER stage_members_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON stage_members
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('stage');

CREATE TRIGGER tasks_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON tasks
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('task');

CREATE TRIGGER bugs_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON bugs
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('bug');

CREATE TRIGGER comments_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON comments
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('comment');

CREATE TRIGGER project_status_changes_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON project_status_changes
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('status_change');

CREATE TRIGGER stage_status_changes_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON stage_status_changes
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('status_change');

CREATE TRIGGER task_status_changes_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON task_status_changes
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('status_change');

CREATE TRIGGER bug_status_changes_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON bug_status_changes
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('status_change');

CREATE TRIGGER stage_changes_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON stage_changes
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('status_change');

CREATE TRIGGER task_changes_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON task_changes
    FOR EACH ROW EXECUTE FUNCTION notify_pmt_change('status_change');

-- +goose Down
DROP TRIGGER projects_notify_change ON projects;
DROP TRIGGER stages_notify_change ON stages;
DROP TRIGGER stage_members_notify_change ON stage_members;
DROP TRIGGER tasks_notify_change ON tasks;
DROP TRIGGER bugs_notify_change ON bugs;
DROP TRIGGER comments_notify_change ON comments;
DROP TRIGGER project_status_changes_notify_change ON project_status_changes;
DROP TRIGGER stage_status_changes_notify_change ON stage_status_changes;
DROP TRIGGER task_status_changes_notify_change ON task_status_changes;
DROP TRIGGER bug_status_changes_notify_change ON bug_status_changes;
DROP TRIGGER stage_changes_notify_change ON stage_changes;
DROP TRIGGER task_changes_notify_change ON task_changes;

DROP FUNCTION notify_pmt_change();
