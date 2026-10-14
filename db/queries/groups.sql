-- name: GetGroupByID :one
SELECT id, owner_id, group_name, group_description, created_at, updated_at
FROM groups
WHERE id = $1;

-- name: ListGroupsByOwner :many
SELECT id, owner_id, group_name, group_description, created_at, updated_at
FROM groups
WHERE owner_id = $1
ORDER BY group_name;

-- name: ListGroupsByMember :many
SELECT g.id, g.owner_id, g.group_name, g.group_description, g.created_at, g.updated_at
FROM groups g
JOIN group_members gm ON gm.group_id = g.id
WHERE gm.user_id = $1
ORDER BY g.group_name;

-- name: CreateGroup :one
INSERT INTO groups (owner_id, group_name, group_description)
VALUES (sqlc.arg(owner_id), sqlc.arg(group_name), sqlc.narg(group_description))
RETURNING id, owner_id, group_name, group_description, created_at, updated_at;

-- name: UpdateGroup :one
UPDATE groups
SET group_name = sqlc.arg(group_name),
    group_description = sqlc.narg(group_description),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id)
RETURNING id, owner_id, group_name, group_description, created_at, updated_at;

-- name: DeleteGroup :exec
DELETE FROM groups
WHERE id = $1 AND owner_id = $2;