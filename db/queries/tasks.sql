-- name: GetTaskByID :one
-- Visible para el creador o el asignado.
SELECT id, user_id, folder_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE tasks.id = $1
  AND (tasks.user_id = $2
       OR tasks.assigned_to = $2);

-- name: ListTasksByUser :many
SELECT id, user_id, folder_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(task_status)::varchar IS NULL OR task_status = sqlc.narg(task_status))
  AND (sqlc.narg(task_priority)::varchar IS NULL OR task_priority = sqlc.narg(task_priority))
  AND (sqlc.narg(folder_id)::integer IS NULL OR folder_id = sqlc.narg(folder_id))
ORDER BY due_date NULLS LAST, id;

-- name: ListTasksByFolder :many
SELECT id, user_id, folder_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE tasks.folder_id = $1 AND tasks.user_id = $2
ORDER BY due_date NULLS LAST, id;

-- name: ListTasksAssignedToUser :many
SELECT id, user_id, folder_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE assigned_to = $1
ORDER BY due_date NULLS LAST, id;

-- name: ListOverdueTasksByUser :many
SELECT id, user_id, folder_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE user_id = $1
  AND task_status <> 'completada'
  AND due_date < CURRENT_TIMESTAMP
ORDER BY due_date;

-- name: CreateTask :one
-- Valida que la carpeta sea del usuario y que el asignado sea el propio usuario.
-- Si algo no cumple, no inserta (la query no devuelve filas).
INSERT INTO tasks (
    user_id, folder_id, assigned_to, title, task_description,
    due_date, task_priority, attachment_path
)
SELECT
    sqlc.arg(user_id)::integer,
    sqlc.narg(folder_id)::integer,
    sqlc.narg(assigned_to)::integer,
    sqlc.arg(title)::varchar,
    sqlc.narg(task_description)::text,
    sqlc.narg(due_date)::timestamptz,
    COALESCE(sqlc.narg(task_priority)::varchar, 'media'),
    sqlc.narg(attachment_path)::varchar
WHERE (sqlc.narg(folder_id)::integer IS NULL
       OR EXISTS (SELECT 1 FROM folders f
                  WHERE f.id = sqlc.narg(folder_id) AND f.user_id = sqlc.arg(user_id)))
  AND (sqlc.narg(assigned_to)::integer IS NULL
       OR sqlc.narg(assigned_to) = sqlc.arg(user_id))
RETURNING id, user_id, folder_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: UpdateTask :one
UPDATE tasks
SET title = $1,
    task_description = $2,
    due_date = $3,
    task_priority = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE tasks.id = $5 AND tasks.user_id = $6
RETURNING id, user_id, folder_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: UpdateTaskStatus :one
-- Puede cambiarlo el creador o el asignado.
UPDATE tasks
SET task_status = $1,
    completed_at = CASE
        WHEN $1::varchar = 'completada' THEN COALESCE(completed_at, CURRENT_TIMESTAMP)
        ELSE NULL
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE tasks.id = $2
  AND (tasks.user_id = $3
       OR tasks.assigned_to = $3)
RETURNING id, user_id, folder_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: MoveTaskToFolder :one
-- Solo el creador, y la carpeta destino (si no es NULL) debe ser suya.
UPDATE tasks
SET folder_id = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE tasks.id = $2
  AND tasks.user_id = $3
  AND ($1::integer IS NULL
       OR EXISTS (SELECT 1 FROM folders f
                  WHERE f.id = $1 AND f.user_id = $3))
RETURNING id, user_id, folder_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: AssignTask :one
-- Solo el creador. El asignado (si no es NULL) debe ser el propio creador.
UPDATE tasks
SET assigned_to = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE tasks.id = $2
  AND tasks.user_id = $3
  AND ($1::integer IS NULL
       OR $1 = $3)
RETURNING id, user_id, folder_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: UpdateTaskAttachment :one
UPDATE tasks
SET attachment_path = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE tasks.id = $2 AND tasks.user_id = $3
RETURNING id, user_id, folder_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1 AND user_id = $2;
