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
JOIN groups_members gm ON gm.group_id = g.id
WHERE gm.user_id = $1
ORDER BY g.group_name;

-- name: CreateGroup :one
WITH new_group AS (
    INSERT INTO groups (owner_id, group_name, group_description)
    VALUES ($1, $2, $3)
    RETURNING id, owner_id, group_name, group_description, created_at, updated_at
),
owner_member AS (
    INSERT INTO groups_members (group_id, user_id)
    SELECT id, owner_id FROM new_group
)
SELECT id, owner_id, group_name, group_description, created_at, updated_at
FROM new_group;

-- name: UpdateGroup :one
UPDATE groups
SET group_name = $1,
    group_description = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $3 AND owner_id = $4
RETURNING id, owner_id, group_name, group_description, created_at, updated_at;

-- name: DeleteGroup :exec
DELETE FROM groups
WHERE id = $1 AND owner_id = $2;
