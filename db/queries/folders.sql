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
SET folder_name = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2 AND user_id = $3
RETURNING id, user_id, folder_name, created_at, updated_at;

-- name: DeleteFolder :exec
DELETE FROM folders
WHERE id = $1 AND user_id = $2;