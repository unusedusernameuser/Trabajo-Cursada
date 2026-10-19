-- name: AddGroupMember :one
INSERT INTO groups_members (group_id, user_id)
VALUES ($1, $2)
RETURNING group_id, user_id, joined_at;

-- name: RemoveGroupMember :exec
DELETE FROM groups_members
WHERE group_id = $1
  AND user_id = $2
  AND user_id <> (SELECT owner_id FROM groups WHERE id = $1);

-- name: IsGroupMember :one
SELECT EXISTS (
    SELECT 1
    FROM groups_members
    WHERE group_id = $1 AND user_id = $2
);

-- name: ListGroupMembers :many
SELECT u.id, u.handle, u.display_name, u.email, gm.joined_at
FROM groups_members gm
JOIN users u ON u.id = gm.user_id
WHERE gm.group_id = $1
ORDER BY u.handle;

-- name: CountGroupMembers :one
SELECT COUNT(*)
FROM groups_members
WHERE group_id = $1;
