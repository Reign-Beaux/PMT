-- +goose Up

-- Bloqueo por persona: una etapa iniciada o un bug en corrección quedan tomados
-- por el usuario que los movió. Sus agentes, con cualquier perfil, escriben en
-- ellos; los de otra persona solo leen. taken_by_name copia el username, que no
-- cambia, para mostrarlo sin leer la tabla de usuarios. Sin llave foránea, como
-- el resto de las referencias a quien actúa: un usuario no se borra.
ALTER TABLE stages
    ADD COLUMN taken_by      UUID,
    ADD COLUMN taken_by_name TEXT,
    ADD COLUMN taken_at      TIMESTAMPTZ,
    ADD CONSTRAINT stages_hold_check CHECK (
        (taken_by IS NULL AND taken_by_name IS NULL AND taken_at IS NULL)
        OR (taken_by IS NOT NULL AND taken_by_name IS NOT NULL AND taken_at IS NOT NULL)
    );

ALTER TABLE bugs
    ADD COLUMN taken_by      UUID,
    ADD COLUMN taken_by_name TEXT,
    ADD COLUMN taken_at      TIMESTAMPTZ,
    ADD CONSTRAINT bugs_hold_check CHECK (
        (taken_by IS NULL AND taken_by_name IS NULL AND taken_at IS NULL)
        OR (taken_by IS NOT NULL AND taken_by_name IS NOT NULL AND taken_at IS NOT NULL)
    );

-- Lo que ya se trabaja queda tomado por el dueño de la instancia. Si todavía no
-- hay usuarios, lo toma al arrancar el servidor, después de crear al dueño.
UPDATE stages s
SET taken_by = u.id, taken_by_name = u.username, taken_at = now()
FROM users u
WHERE u.is_instance_owner AND s.retired_at IS NULL AND s.status IN ('in_dev', 'stopped');

UPDATE bugs b
SET taken_by = u.id, taken_by_name = u.username, taken_at = now()
FROM users u
WHERE u.is_instance_owner
  AND b.status IN ('in_dev', 'in_review', 'changes_requested', 'in_qa', 'pending_close');

-- pmt_hold_of da quién tiene tomado lo que contiene a un elemento: la etapa de
-- una etapa, tarea, plan QAP, plan de desarrollo de etapa o hallazgo de tarea, y
-- el bug de un bug, plan de desarrollo de bug o hallazgo de bug. holder_kind es
-- stage o bug; sin filas si el elemento no existe o no pertenece a ninguno (un
-- proyecto, un comentario del proyecto). La usa la comprobación de escrituras.
-- +goose StatementBegin
CREATE FUNCTION pmt_hold_of(kind TEXT, item UUID)
RETURNS TABLE (holder_kind TEXT, taken_by UUID, taken_by_name TEXT) AS $$
BEGIN
    CASE kind
    WHEN 'stage' THEN
        RETURN QUERY SELECT 'stage'::text, s.taken_by, s.taken_by_name FROM stages s WHERE s.id = item;
    WHEN 'task' THEN
        RETURN QUERY SELECT 'stage'::text, s.taken_by, s.taken_by_name
            FROM tasks t JOIN stages s ON s.id = t.stage_id WHERE t.id = item;
    WHEN 'bug' THEN
        RETURN QUERY SELECT 'bug'::text, b.taken_by, b.taken_by_name FROM bugs b WHERE b.id = item;
    WHEN 'finding' THEN
        RETURN QUERY SELECT h.* FROM findings f, pmt_hold_of(f.item_kind, f.item_id) h WHERE f.id = item;
    WHEN 'test_plan' THEN
        RETURN QUERY SELECT 'stage'::text, s.taken_by, s.taken_by_name
            FROM test_plans p JOIN stages s ON s.id = p.stage_id WHERE p.id = item;
    WHEN 'dev_plan' THEN
        RETURN QUERY
            SELECT CASE WHEN d.stage_id IS NOT NULL THEN 'stage' ELSE 'bug' END::text,
                   COALESCE(s.taken_by, b.taken_by), COALESCE(s.taken_by_name, b.taken_by_name)
            FROM dev_plans d
            LEFT JOIN stages s ON s.id = d.stage_id
            LEFT JOIN bugs b ON b.id = d.bug_id
            WHERE d.id = item;
    ELSE
        RETURN;
    END CASE;
END;
$$ LANGUAGE plpgsql STABLE;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION pmt_hold_of(TEXT, UUID);
ALTER TABLE bugs
    DROP CONSTRAINT bugs_hold_check,
    DROP COLUMN taken_at,
    DROP COLUMN taken_by_name,
    DROP COLUMN taken_by;
ALTER TABLE stages
    DROP CONSTRAINT stages_hold_check,
    DROP COLUMN taken_at,
    DROP COLUMN taken_by_name,
    DROP COLUMN taken_by;
