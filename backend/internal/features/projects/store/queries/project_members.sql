-- name: RoleOf :one
-- El proyecto de un elemento y el rol del usuario en él. Sin filas si el
-- elemento no existe; rol vacío si el usuario no es miembro.
SELECT target.project_id::uuid AS project_id, COALESCE(m.role, '')::text AS role
FROM (SELECT pmt_project_of(@kind::text, @item_id::uuid) AS project_id) target
LEFT JOIN project_members m ON m.project_id = target.project_id AND m.user_id = @user_id::uuid
WHERE target.project_id IS NOT NULL;

-- name: ListProjectMembers :many
SELECT * FROM project_members
WHERE project_id = $1
ORDER BY CASE role WHEN 'owner' THEN 0 WHEN 'editor' THEN 1 ELSE 2 END, added_at, user_id;

-- name: GetProjectMember :one
SELECT * FROM project_members
WHERE project_id = $1 AND user_id = $2;

-- name: InsertProjectMember :exec
INSERT INTO project_members (project_id, user_id, role)
VALUES ($1, $2, $3);

-- name: UpdateProjectMemberRole :execrows
UPDATE project_members SET role = $3
WHERE project_id = $1 AND user_id = $2 AND role <> 'owner';

-- name: DeleteProjectMember :execrows
DELETE FROM project_members
WHERE project_id = $1 AND user_id = $2 AND role <> 'owner';

-- name: AdoptOrphanProjects :execrows
-- Deja al usuario como owner de los proyectos sin owner.
INSERT INTO project_members (project_id, user_id, role)
SELECT p.id, @user_id, 'owner' FROM projects p
WHERE NOT EXISTS (SELECT 1 FROM project_members m WHERE m.project_id = p.id AND m.role = 'owner')
ON CONFLICT (project_id, user_id) DO UPDATE SET role = 'owner';
