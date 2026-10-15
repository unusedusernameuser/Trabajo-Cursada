-- name: GetTagByID :one
SELECT id, user_id, tag_name, created_at, updated_at
FROM tags
WHERE id = $1 AND user_id = $2;

-- name: ListTagsByUser :many
SELECT id, user_id, tag_name, created_at, updated_at
FROM tags
WHERE user_id = $1
ORDER BY tag_name;

-- name: CreateTag :one
INSERT INTO tags (user_id, tag_name)
VALUES ($1, $2)
RETURNING id, user_id, tag_name, created_at, updated_at;

-- name: UpdateTagName :one
UPDATE tags
SET tag_name = sqlc.arg(tag_name),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING id, user_id, tag_name, created_at, updated_at;

-- name: DeleteTag :exec
DELETE FROM tags
WHERE id = $1 AND user_id = $2;
