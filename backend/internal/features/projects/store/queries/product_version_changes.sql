-- name: CreateProductVersionChange :exec
INSERT INTO product_version_changes (id, project_id, from_version, to_version, actor_name, actor_profile, reason)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListProductVersionChanges :many
SELECT * FROM product_version_changes
WHERE project_id = $1
ORDER BY changed_at, id;
