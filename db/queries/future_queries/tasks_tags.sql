-- name: AddTagToTask :exec
INSERT INTO tasks_tags (task_id, tag_id)
SELECT t.id, tg.id
FROM tasks t
JOIN tags tg ON tg.user_id = t.user_id
WHERE t.id = sqlc.arg(task_id)
  AND tg.id = sqlc.arg(tag_id)
  AND t.user_id = sqlc.arg(user_id)
ON CONFLICT (task_id, tag_id) DO NOTHING;

-- name: RemoveTagFromTask :exec
DELETE FROM tasks_tags tt
USING tasks t
WHERE tt.task_id = t.id
  AND tt.task_id = sqlc.arg(task_id)
  AND tt.tag_id = sqlc.arg(tag_id)
  AND t.user_id = sqlc.arg(user_id);

-- name: ClearTaskTags :exec
DELETE FROM tasks_tags tt
USING tasks t
WHERE tt.task_id = t.id
  AND tt.task_id = sqlc.arg(task_id)
  AND t.user_id = sqlc.arg(user_id);

-- name: ListTagsByTask :many
SELECT tg.id, tg.user_id, tg.tag_name, tg.created_at, tg.updated_at
FROM tags tg
JOIN tasks_tags tt ON tt.tag_id = tg.id
WHERE tt.task_id = sqlc.arg(task_id)
  AND tg.user_id = sqlc.arg(user_id)
ORDER BY tg.tag_name;

-- name: ListTasksByTag :many
SELECT t.id, t.user_id, t.folder_id, t.group_id, t.assigned_to, t.title, t.task_description,
       t.due_date, t.task_status, t.task_priority, t.attachment_path,
       t.created_at, t.updated_at, t.completed_at
FROM tasks t
JOIN tasks_tags tt ON tt.task_id = t.id
JOIN tags tg ON tg.id = tt.tag_id
WHERE tt.tag_id = sqlc.arg(tag_id)
  AND tg.user_id = sqlc.arg(user_id)
ORDER BY t.due_date NULLS LAST, t.id;
