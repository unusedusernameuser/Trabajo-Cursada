-- name: GetUserByID :one
SELECT id, handle, display_name, email, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByHandle :one
SELECT id, handle, display_name, email, created_at, updated_at
FROM users
WHERE handle = LOWER(sqlc.arg(handle));

-- name: GetUserByEmail :one
SELECT id, handle, display_name, email, created_at, updated_at
FROM users
WHERE email = LOWER(sqlc.arg(email));

-- name: GetUserAuthByEmail :one
SELECT id, password_hash
FROM users
WHERE email = LOWER(sqlc.arg(email));

-- name: ListUsers :many
SELECT id, handle, display_name, email, created_at, updated_at
FROM users
ORDER BY handle;

-- name: CreateUser :one
INSERT INTO users (handle, display_name, email, password_hash)
VALUES (LOWER(sqlc.arg(handle)), sqlc.arg(display_name), LOWER(sqlc.arg(email)), sqlc.arg(password_hash))
RETURNING id, handle, display_name, email, created_at, updated_at;

-- name: UpdateDisplayName :one
UPDATE users
SET display_name = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, handle, display_name, email, created_at, updated_at;

-- name: UpdateHandle :one
UPDATE users
SET handle = LOWER(sqlc.arg(handle)),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING id, handle, display_name, email, created_at, updated_at;

-- name: UpdateEmail :one
UPDATE users
SET email = LOWER(sqlc.arg(email)),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING id, handle, display_name, email, created_at, updated_at;

-- name: UpdatePassword :exec
UPDATE users
SET password_hash = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;