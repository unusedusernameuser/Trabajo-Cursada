-- name: GetTaskByID :one
-- Solo el dueño de la tarea puede verla.
SELECT id, user_id, folder_id, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE tasks.id = sqlc.arg(id) AND tasks.user_id = sqlc.arg(user_id);

-- name: ListTasksByUser :many
SELECT id, user_id, folder_id, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(task_status)::varchar IS NULL OR task_status = sqlc.narg(task_status))
  AND (sqlc.narg(task_priority)::varchar IS NULL OR task_priority = sqlc.narg(task_priority))
  AND (sqlc.narg(folder_id)::integer IS NULL OR folder_id = sqlc.narg(folder_id))
ORDER BY due_date NULLS LAST, id;

-- name: ListTasksByFolder :many
SELECT id, user_id, folder_id, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE tasks.folder_id = sqlc.arg(folder_id) AND tasks.user_id = sqlc.arg(user_id)
ORDER BY due_date NULLS LAST, id;

-- name: ListOverdueTasksByUser :many
SELECT id, user_id, folder_id, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE user_id = $1
  AND task_status <> 'completada'
  AND due_date < CURRENT_TIMESTAMP
ORDER BY due_date;

-- name: CreateTask :one
-- Valida que la carpeta (si viene) sea del usuario. Si no lo es, no inserta
-- (la query no devuelve filas).
INSERT INTO tasks (
    user_id, folder_id, title, task_description,
    due_date, task_priority, attachment_path
)
SELECT
    sqlc.arg(user_id)::integer,
    sqlc.narg(folder_id)::integer,
    sqlc.arg(title)::varchar,
    sqlc.narg(task_description)::text,
    sqlc.narg(due_date)::timestamptz,
    COALESCE(sqlc.narg(task_priority)::varchar, 'media'),
    sqlc.narg(attachment_path)::varchar
WHERE (sqlc.narg(folder_id)::integer IS NULL
       OR EXISTS (SELECT 1 FROM folders f
                  WHERE f.id = sqlc.narg(folder_id) AND f.user_id = sqlc.arg(user_id)))
RETURNING id, user_id, folder_id, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: UpdateTask :one
UPDATE tasks
SET title = sqlc.arg(title),
    task_description = sqlc.narg(task_description),
    due_date = sqlc.narg(due_date),
    task_priority = sqlc.arg(task_priority),
    updated_at = CURRENT_TIMESTAMP
WHERE tasks.id = sqlc.arg(id) AND tasks.user_id = sqlc.arg(user_id)
RETURNING id, user_id, folder_id, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: UpdateTaskStatus :one
UPDATE tasks
SET task_status = sqlc.arg(task_status)::varchar,
    completed_at = CASE
        WHEN sqlc.arg(task_status)::varchar = 'completada' THEN COALESCE(completed_at, CURRENT_TIMESTAMP)
        ELSE NULL
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE tasks.id = sqlc.arg(id) AND tasks.user_id = sqlc.arg(user_id)
RETURNING id, user_id, folder_id, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: MoveTaskToFolder :one
-- Solo el creador, y la carpeta destino (si no es NULL) debe ser suya.
UPDATE tasks
SET folder_id = sqlc.narg(folder_id),
    updated_at = CURRENT_TIMESTAMP
WHERE tasks.id = sqlc.arg(id)
  AND tasks.user_id = sqlc.arg(user_id)
  AND (sqlc.narg(folder_id)::integer IS NULL
       OR EXISTS (SELECT 1 FROM folders f
                  WHERE f.id = sqlc.narg(folder_id) AND f.user_id = sqlc.arg(user_id)))
RETURNING id, user_id, folder_id, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: UpdateTaskAttachment :one
UPDATE tasks
SET attachment_path = sqlc.narg(attachment_path),
    updated_at = CURRENT_TIMESTAMP
WHERE tasks.id = sqlc.arg(id) AND tasks.user_id = sqlc.arg(user_id)
RETURNING id, user_id, folder_id, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1 AND user_id = $2;
