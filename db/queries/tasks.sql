-- name: GetTaskByID :one
SELECT id, user_id, folder_id, group_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE id = $1;

-- name: ListTasksByUser :many
SELECT id, user_id, folder_id, group_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(task_status)::varchar IS NULL OR task_status = sqlc.narg(task_status))
  AND (sqlc.narg(task_priority)::varchar IS NULL OR task_priority = sqlc.narg(task_priority))
  AND (sqlc.narg(folder_id)::integer IS NULL OR folder_id = sqlc.narg(folder_id))
ORDER BY due_date NULLS LAST, id;

-- name: ListTasksByFolder :many
SELECT id, user_id, folder_id, group_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE folder_id = $1
ORDER BY due_date NULLS LAST, id;

-- name: ListTasksByGroup :many
SELECT id, user_id, folder_id, group_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE group_id = $1
ORDER BY due_date NULLS LAST, id;

-- name: ListTasksAssignedToUser :many
SELECT id, user_id, folder_id, group_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE assigned_to = $1
ORDER BY due_date NULLS LAST, id;

-- name: ListOverdueTasksByUser :many
SELECT id, user_id, folder_id, group_id, assigned_to, title, task_description,
       due_date, task_status, task_priority, attachment_path,
       created_at, updated_at, completed_at
FROM tasks
WHERE user_id = $1
  AND task_status <> 'completada'
  AND due_date < CURRENT_TIMESTAMP
ORDER BY due_date;

-- name: CreateTask :one
INSERT INTO tasks (
    user_id, folder_id, group_id, assigned_to, title, task_description,
    due_date, task_priority, attachment_path
)
VALUES (
    sqlc.arg(user_id),
    sqlc.narg(folder_id),
    sqlc.narg(group_id),
    sqlc.narg(assigned_to),
    sqlc.arg(title),
    sqlc.narg(task_description),
    sqlc.narg(due_date),
    COALESCE(sqlc.narg(task_priority)::varchar, 'media'),
    sqlc.narg(attachment_path)
)
RETURNING id, user_id, folder_id, group_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: UpdateTask :one
UPDATE tasks
SET title = sqlc.arg(title),
    task_description = sqlc.narg(task_description),
    due_date = sqlc.narg(due_date),
    task_priority = sqlc.arg(task_priority),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING id, user_id, folder_id, group_id, assigned_to, title, task_description,
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
WHERE id = sqlc.arg(id)
RETURNING id, user_id, folder_id, group_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: MoveTaskToFolder :one
UPDATE tasks
SET folder_id = sqlc.narg(folder_id),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING id, user_id, folder_id, group_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: AssignTask :one
UPDATE tasks
SET assigned_to = sqlc.narg(assigned_to),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING id, user_id, folder_id, group_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: UpdateTaskAttachment :one
UPDATE tasks
SET attachment_path = sqlc.narg(attachment_path),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING id, user_id, folder_id, group_id, assigned_to, title, task_description,
          due_date, task_status, task_priority, attachment_path,
          created_at, updated_at, completed_at;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1 AND user_id = $2;