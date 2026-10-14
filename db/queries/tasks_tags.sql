-- name: AddTagToTask :exec
-- Solo asocia si la etiqueta pertenece al mismo usuario dueño de la tarea.
INSERT INTO task_tag (task_id, tag_id)
SELECT t.id, tg.id
FROM tasks t
JOIN tags tg ON tg.user_id = t.user_id
WHERE t.id = $1 AND tg.id = $2
ON CONFLICT (task_id, tag_id) DO NOTHING;

-- name: RemoveTagFromTask :exec
DELETE FROM task_tag
WHERE task_id = $1 AND tag_id = $2;

-- name: ClearTaskTags :exec
DELETE FROM task_tag
WHERE task_id = $1;

-- name: ListTagsByTask :many
SELECT tg.id, tg.user_id, tg.tag_name, tg.created_at, tg.updated_at
FROM tags tg
JOIN task_tag tt ON tt.tag_id = tg.id
WHERE tt.task_id = $1
ORDER BY tg.tag_name;

-- name: ListTasksByTag :many
SELECT t.id, t.user_id, t.folder_id, t.group_id, t.assigned_to, t.title, t.task_description,
       t.due_date, t.task_status, t.task_priority, t.attachment_path,
       t.created_at, t.updated_at, t.completed_at
FROM tasks t
JOIN task_tag tt ON tt.task_id = t.id
WHERE tt.tag_id = $1
ORDER BY t.due_date NULLS LAST, t.id;