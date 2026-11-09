-- name: GetFolderByID :one
SELECT id, user_id, folder_name, created_at, updated_at
FROM folders
WHERE id = $1 AND user_id = $2;

-- name: ListFoldersByUser :many
SELECT id, user_id, folder_name, created_at, updated_at
FROM folders
WHERE user_id = $1
ORDER BY folder_name;

-- name: CreateFolder :one
INSERT INTO folders (user_id, folder_name)
VALUES ($1, $2)
RETURNING id, user_id, folder_name, created_at, updated_at;

-- name: UpdateFolderName :one
UPDATE folders
SET folder_name = sqlc.arg(folder_name),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING id, user_id, folder_name, created_at, updated_at;

-- name: DeleteFolder :exec
DELETE FROM folders
WHERE id = $1 AND user_id = $2;