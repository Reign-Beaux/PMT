#!/usr/bin/env bash
# Deja a un usuario como owner de todos los proyectos de PMT. El owner anterior
# de cada proyecto queda como editor; los demás miembros no cambian.
set -euo pipefail
usage() { echo 'uso: owner.sh <username>' >&2; exit 2; }
(($# == 1)) || usage
username=$1
container=${BACKUP_CONTAINER:-}
db_user=${BACKUP_DB_USER:-pmt}
db=${BACKUP_DB:-pmt}
[[ -n $container ]] || { echo 'owner: falta BACKUP_CONTAINER' >&2; exit 1; }
[[ $username =~ ^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$ ]] || { echo "owner: username no válido: $username" >&2; exit 1; }

docker exec -i "$container" psql -U "$db_user" -d "$db" -v ON_ERROR_STOP=1 -qtA -v username="$username" <<'SQL'
BEGIN;
CREATE TEMP TABLE target ON COMMIT DROP AS
SELECT id FROM users WHERE username = :'username' AND disabled_at IS NULL;
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM target) THEN
        RAISE EXCEPTION 'no existe un usuario activo con ese username';
    END IF;
END $$;
-- Primero se baja al owner anterior: solo puede haber un owner por proyecto.
UPDATE project_members m SET role = 'editor'
WHERE m.role = 'owner' AND m.user_id <> (SELECT id FROM target);
INSERT INTO project_members (project_id, user_id, role)
SELECT p.id, (SELECT id FROM target), 'owner' FROM projects p
ON CONFLICT (project_id, user_id) DO UPDATE SET role = 'owner';
SELECT 'proyectos de los que es owner: ' || count(*)
FROM project_members WHERE role = 'owner' AND user_id = (SELECT id FROM target);
COMMIT;
SQL
