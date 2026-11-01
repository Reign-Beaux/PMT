-- +goose Up
-- Los registros de *_status_changes conservan in_progress: son inmutables y ese
-- era el valor real al escribirlos. updated_at no cambia: no es un cambio de negocio.
ALTER TABLE tasks DROP CONSTRAINT tasks_status_check;
UPDATE tasks SET status = 'in_dev' WHERE status = 'in_progress';
ALTER TABLE tasks ADD CONSTRAINT tasks_status_check CHECK (status IN (
    'backlog', 'ready', 'in_dev', 'in_review', 'changes_requested',
    'in_qa', 'pending_close', 'done', 'cancelled'
));

ALTER TABLE bugs DROP CONSTRAINT bugs_status_check;
UPDATE bugs SET status = 'in_dev' WHERE status = 'in_progress';
ALTER TABLE bugs ADD CONSTRAINT bugs_status_check CHECK (status IN (
    'reported', 'confirmed', 'in_dev', 'in_review', 'changes_requested',
    'in_qa', 'pending_close', 'done', 'rejected'
));

-- +goose Down
-- Falla si existen bugs en 'pending_close', porque el CHECK anterior no los admite.
ALTER TABLE tasks DROP CONSTRAINT tasks_status_check;
UPDATE tasks SET status = 'in_progress' WHERE status = 'in_dev';
ALTER TABLE tasks ADD CONSTRAINT tasks_status_check CHECK (status IN (
    'backlog', 'ready', 'in_progress', 'in_review', 'changes_requested',
    'in_qa', 'pending_close', 'done', 'cancelled'
));

ALTER TABLE bugs DROP CONSTRAINT bugs_status_check;
UPDATE bugs SET status = 'in_progress' WHERE status = 'in_dev';
ALTER TABLE bugs ADD CONSTRAINT bugs_status_check CHECK (status IN (
    'reported', 'confirmed', 'in_progress', 'in_review', 'changes_requested',
    'in_qa', 'done', 'rejected'
));
