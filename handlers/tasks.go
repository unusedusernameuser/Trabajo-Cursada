package main

import (
	"net/http"
	"slices"
	"time"

	db "my_sqlc_project/db/sqlc"
)

// taskHandler agrupa los handlers de tasks y tasks_tags.
// Todas las rutas cuelgan de /users/{user_id}: es quien consulta o modifica la tarea.
type taskHandler struct {
	queries *db.Queries
}

var (
	validStatuses   = []string{"pendiente", "en_progreso", "completada"}
	validPriorities = []string{"baja", "media", "alta"}
)

// taskInput es el cuerpo de POST /users/{user_id}/tasks.
type taskInput struct {
	FolderID        *int32     `json:"folder_id"`
	GroupID         *int32     `json:"group_id"`
	AssignedTo      *int32     `json:"assigned_to"`
	Title           string     `json:"title"`
	TaskDescription *string    `json:"task_description"`
	DueDate         *time.Time `json:"due_date"` // RFC 3339, ej. "2026-12-01T18:00:00Z"
	TaskPriority    *string    `json:"task_priority"`
	AttachmentPath  *string    `json:"attachment_path"`
}

// updateTaskInput es el cuerpo de PUT /users/{user_id}/tasks/{id}.
type updateTaskInput struct {
	Title           string     `json:"title"`
	TaskDescription *string    `json:"task_description"`
	DueDate         *time.Time `json:"due_date"`
	TaskPriority    string     `json:"task_priority"`
}

type taskResponse struct {
	ID              int32      `json:"id"`
	UserID          int32      `json:"user_id"`
	FolderID        *int32     `json:"folder_id"`
	GroupID         *int32     `json:"group_id"`
	AssignedTo      *int32     `json:"assigned_to"`
	Title           string     `json:"title"`
	TaskDescription *string    `json:"task_description"`
	DueDate         *time.Time `json:"due_date"`
	TaskStatus      string     `json:"task_status"`
	TaskPriority    string     `json:"task_priority"`
	AttachmentPath  *string    `json:"attachment_path"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	CompletedAt     *time.Time `json:"completed_at"`
}

func newTaskResponse(t db.Task) taskResponse {
	return taskResponse{
		ID:              t.ID,
		UserID:          t.UserID,
		FolderID:        nullInt32(t.FolderID),
		GroupID:         nullInt32(t.GroupID),
		AssignedTo:      nullInt32(t.AssignedTo),
		Title:           t.Title,
		TaskDescription: nullString(t.TaskDescription),
		DueDate:         nullTime(t.DueDate),
		TaskStatus:      t.TaskStatus,
		TaskPriority:    t.TaskPriority,
		AttachmentPath:  nullString(t.AttachmentPath),
		CreatedAt:       nullTime(t.CreatedAt),
		UpdatedAt:       nullTime(t.UpdatedAt),
		CompletedAt:     nullTime(t.CompletedAt),
	}
}

func registerTaskRoutes(queries *db.Queries) {
	h := &taskHandler{queries: queries}

	// CRUD y listados de tareas
	http.HandleFunc("POST /users/{user_id}/tasks", h.create)
	http.HandleFunc("GET /users/{user_id}/tasks", h.list) // ?status=&priority=&folder_id=
	http.HandleFunc("GET /users/{user_id}/tasks/overdue", h.listOverdue)
	http.HandleFunc("GET /users/{user_id}/tasks/{id}", h.get)
	http.HandleFunc("PUT /users/{user_id}/tasks/{id}", h.update)
	http.HandleFunc("DELETE /users/{user_id}/tasks/{id}", h.delete)
	http.HandleFunc("GET /users/{user_id}/assigned-tasks", h.listAssigned)
	http.HandleFunc("GET /users/{user_id}/folders/{folder_id}/tasks", h.listByFolder)
	http.HandleFunc("GET /users/{user_id}/groups/{group_id}/tasks", h.listByGroup)

	// Cambios parciales
	http.HandleFunc("PUT /users/{user_id}/tasks/{id}/status", h.updateStatus)
	http.HandleFunc("PUT /users/{user_id}/tasks/{id}/folder", h.moveToFolder)
	http.HandleFunc("PUT /users/{user_id}/tasks/{id}/assignee", h.assign)
	http.HandleFunc("PUT /users/{user_id}/tasks/{id}/attachment", h.updateAttachment)

	// Tags de una tarea
	http.HandleFunc("GET /users/{user_id}/tasks/{id}/tags", h.listTags)
	http.HandleFunc("PUT /users/{user_id}/tasks/{id}/tags/{tag_id}", h.addTag)
	http.HandleFunc("DELETE /users/{user_id}/tasks/{id}/tags/{tag_id}", h.removeTag)
	http.HandleFunc("DELETE /users/{user_id}/tasks/{id}/tags", h.clearTags)
	http.HandleFunc("GET /users/{user_id}/tags/{tag_id}/tasks", h.listByTag)
}

func writeTasks(w http.ResponseWriter, tasks []db.Task) {
	out := make([]taskResponse, 0, len(tasks)) // para responder [] y no null
	for _, t := range tasks {
		out = append(out, newTaskResponse(t))
	}
	writeJSON(w, http.StatusOK, out)
}

// checkPriority valida que la prioridad sea una de las permitidas por el schema.
func checkPriority(w http.ResponseWriter, p string) bool {
	if !slices.Contains(validPriorities, p) {
		writeError(w, http.StatusBadRequest, "task_priority debe ser baja, media o alta")
		return false
	}
	return true
}

// checkAttachment valida attachment_path (si viene): no vacío y hasta 255 caracteres.
func checkAttachment(w http.ResponseWriter, p *string) bool {
	return p == nil || checkText(w, "attachment_path", *p, 255)
}

// POST /users/{user_id}/tasks
func (h *taskHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	var in taskInput
	if !decodeBody(w, r, &in) || !checkText(w, "title", in.Title, 127) {
		return
	}
	if in.TaskPriority != nil && !checkPriority(w, *in.TaskPriority) {
		return
	}
	if !checkAttachment(w, in.AttachmentPath) {
		return
	}

	task, err := h.queries.CreateTask(r.Context(), db.CreateTaskParams{
		UserID:          userID,
		FolderID:        toNullInt32(in.FolderID),
		GroupID:         toNullInt32(in.GroupID),
		AssignedTo:      toNullInt32(in.AssignedTo),
		Title:           in.Title,
		TaskDescription: toNullString(in.TaskDescription),
		DueDate:         toNullTime(in.DueDate),
		TaskPriority:    toNullString(in.TaskPriority),
		AttachmentPath:  toNullString(in.AttachmentPath),
	})
	if err != nil {
		// Si la carpeta no es del usuario, no es miembro del grupo o el asignado no es
		// válido, la query no inserta nada y no devuelve filas.
		if isNoRows(err) {
			writeError(w, http.StatusBadRequest,
				"la carpeta debe ser del usuario, el usuario debe ser miembro del grupo y el asignado debe ser el propio usuario o miembro del grupo")
			return
		}
		writeDBError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newTaskResponse(task))
}

// GET /users/{user_id}/tasks[?status=&priority=&folder_id=]
func (h *taskHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}

	params := db.ListTasksByUserParams{UserID: userID}
	q := r.URL.Query()
	if v := q.Get("status"); v != "" {
		if !slices.Contains(validStatuses, v) {
			writeError(w, http.StatusBadRequest, "status debe ser pendiente, en_progreso o completada")
			return
		}
		params.TaskStatus = toNullString(&v)
	}
	if v := q.Get("priority"); v != "" {
		if !checkPriority(w, v) {
			return
		}
		params.TaskPriority = toNullString(&v)
	}
	if v := q.Get("folder_id"); v != "" {
		id, ok := parsePositiveInt32(v)
		if !ok {
			writeError(w, http.StatusBadRequest, "folder_id debe ser un entero positivo")
			return
		}
		params.FolderID = toNullInt32(&id)
	}

	tasks, err := h.queries.ListTasksByUser(r.Context(), params)
	if err != nil {
		writeDBError(w, err)
		return
	}
	writeTasks(w, tasks)
}

// GET /users/{user_id}/tasks/overdue
func (h *taskHandler) listOverdue(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}

	tasks, err := h.queries.ListOverdueTasksByUser(r.Context(), userID)
	if err != nil {
		writeDBError(w, err)
		return
	}
	writeTasks(w, tasks)
}

// GET /users/{user_id}/assigned-tasks
func (h *taskHandler) listAssigned(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}

	tasks, err := h.queries.ListTasksAssignedToUser(r.Context(), toNullInt32(&userID))
	if err != nil {
		writeDBError(w, err)
		return
	}
	writeTasks(w, tasks)
}

// GET /users/{user_id}/folders/{folder_id}/tasks
func (h *taskHandler) listByFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	folderID, ok := readPathID(w, r, "folder_id")
	if !ok {
		return
	}

	tasks, err := h.queries.ListTasksByFolder(r.Context(), db.ListTasksByFolderParams{
		FolderID: toNullInt32(&folderID),
		UserID:   userID,
	})
	if err != nil {
		writeDBError(w, err)
		return
	}
	writeTasks(w, tasks)
}

// GET /users/{user_id}/groups/{group_id}/tasks
// Solo devuelve tareas si el usuario es miembro del grupo.
func (h *taskHandler) listByGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	groupID, ok := readPathID(w, r, "group_id")
	if !ok {
		return
	}

	tasks, err := h.queries.ListTasksByGroup(r.Context(), db.ListTasksByGroupParams{
		GroupID: toNullInt32(&groupID),
		UserID:  userID,
	})
	if err != nil {
		writeDBError(w, err)
		return
	}
	writeTasks(w, tasks)
}

// GET /users/{user_id}/tasks/{id}
// Visible para el creador, el asignado o cualquier miembro del grupo de la tarea.
func (h *taskHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}

	task, err := h.queries.GetTaskByID(r.Context(), db.GetTaskByIDParams{ID: id, UserID: userID})
	if err != nil {
		writeDBErrorMsg(w, err, "tarea no encontrada", "")
		return
	}

	writeJSON(w, http.StatusOK, newTaskResponse(task))
}

// PUT /users/{user_id}/tasks/{id}
// Solo el creador. Modifica title, task_description, due_date y task_priority.
func (h *taskHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	var in updateTaskInput
	if !decodeBody(w, r, &in) || !checkText(w, "title", in.Title, 127) || !checkPriority(w, in.TaskPriority) {
		return
	}

	task, err := h.queries.UpdateTask(r.Context(), db.UpdateTaskParams{
		ID:              id,
		UserID:          userID,
		Title:           in.Title,
		TaskDescription: toNullString(in.TaskDescription),
		DueDate:         toNullTime(in.DueDate),
		TaskPriority:    in.TaskPriority,
	})
	if err != nil {
		writeDBErrorMsg(w, err, "tarea no encontrada", "")
		return
	}

	writeJSON(w, http.StatusOK, newTaskResponse(task))
}

// DELETE /users/{user_id}/tasks/{id}   (solo el creador)
func (h *taskHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}

	if err := h.queries.DeleteTask(r.Context(), db.DeleteTaskParams{ID: id, UserID: userID}); err != nil {
		writeDBError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PUT /users/{user_id}/tasks/{id}/status   {"task_status": "completada"}
// Puede cambiarlo el creador, el asignado o cualquier miembro del grupo.
// completed_at se completa al pasar a "completada" y vuelve a null con cualquier otro estado.
func (h *taskHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	var in struct {
		TaskStatus string `json:"task_status"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	if !slices.Contains(validStatuses, in.TaskStatus) {
		writeError(w, http.StatusBadRequest, "task_status debe ser pendiente, en_progreso o completada")
		return
	}

	task, err := h.queries.UpdateTaskStatus(r.Context(), db.UpdateTaskStatusParams{
		ID:         id,
		UserID:     userID,
		TaskStatus: in.TaskStatus,
	})
	if err != nil {
		writeDBErrorMsg(w, err, "tarea no encontrada", "")
		return
	}

	writeJSON(w, http.StatusOK, newTaskResponse(task))
}

// PUT /users/{user_id}/tasks/{id}/folder   {"folder_id": 3}  (null la saca de la carpeta)
// Solo el creador, y la carpeta destino debe ser suya.
func (h *taskHandler) moveToFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	var in struct {
		FolderID *int32 `json:"folder_id"`
	}
	if !decodeBody(w, r, &in) {
		return
	}

	task, err := h.queries.MoveTaskToFolder(r.Context(), db.MoveTaskToFolderParams{
		ID:       id,
		UserID:   userID,
		FolderID: toNullInt32(in.FolderID),
	})
	if err != nil {
		writeDBErrorMsg(w, err, "tarea no encontrada o la carpeta no es del usuario", "")
		return
	}

	writeJSON(w, http.StatusOK, newTaskResponse(task))
}

// PUT /users/{user_id}/tasks/{id}/assignee   {"assigned_to": 3}  (null la desasigna)
// Solo el creador. El asignado debe ser el propio creador o miembro del grupo de la tarea.
func (h *taskHandler) assign(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	var in struct {
		AssignedTo *int32 `json:"assigned_to"`
	}
	if !decodeBody(w, r, &in) {
		return
	}

	task, err := h.queries.AssignTask(r.Context(), db.AssignTaskParams{
		ID:         id,
		UserID:     userID,
		AssignedTo: toNullInt32(in.AssignedTo),
	})
	if err != nil {
		writeDBErrorMsg(w, err, "tarea no encontrada o el asignado no es válido", "")
		return
	}

	writeJSON(w, http.StatusOK, newTaskResponse(task))
}

// PUT /users/{user_id}/tasks/{id}/attachment   {"attachment_path": "/ruta"}  (null lo quita)
func (h *taskHandler) updateAttachment(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	var in struct {
		AttachmentPath *string `json:"attachment_path"`
	}
	if !decodeBody(w, r, &in) || !checkAttachment(w, in.AttachmentPath) {
		return
	}

	task, err := h.queries.UpdateTaskAttachment(r.Context(), db.UpdateTaskAttachmentParams{
		ID:             id,
		UserID:         userID,
		AttachmentPath: toNullString(in.AttachmentPath),
	})
	if err != nil {
		writeDBErrorMsg(w, err, "tarea no encontrada", "")
		return
	}

	writeJSON(w, http.StatusOK, newTaskResponse(task))
}

// ---- Tags de una tarea (tasks_tags.sql) ----

// GET /users/{user_id}/tasks/{id}/tags
func (h *taskHandler) listTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}

	tags, err := h.taskTags(r, id, userID)
	if err != nil {
		writeDBError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

// PUT /users/{user_id}/tasks/{id}/tags/{tag_id}
// Asocia el tag a la tarea (idempotente). Tarea y tag deben ser del usuario.
func (h *taskHandler) addTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	tagID, ok := readPathID(w, r, "tag_id")
	if !ok {
		return
	}
	ctx := r.Context()

	// La query no falla si la tarea o el tag no son del usuario: simplemente no inserta.
	// Por eso se verifica el resultado para poder responder 404.
	if err := h.queries.AddTagToTask(ctx, db.AddTagToTaskParams{TaskID: id, TagID: tagID, UserID: userID}); err != nil {
		writeDBError(w, err)
		return
	}
	tags, err := h.taskTags(r, id, userID)
	if err != nil {
		writeDBError(w, err)
		return
	}
	if !slices.ContainsFunc(tags, func(t tagResponse) bool { return t.ID == tagID }) {
		writeError(w, http.StatusNotFound, "la tarea o el tag no existen o no son del usuario")
		return
	}

	writeJSON(w, http.StatusOK, tags)
}

// DELETE /users/{user_id}/tasks/{id}/tags/{tag_id}
func (h *taskHandler) removeTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	tagID, ok := readPathID(w, r, "tag_id")
	if !ok {
		return
	}

	if err := h.queries.RemoveTagFromTask(r.Context(), db.RemoveTagFromTaskParams{TaskID: id, TagID: tagID, UserID: userID}); err != nil {
		writeDBError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /users/{user_id}/tasks/{id}/tags
func (h *taskHandler) clearTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}

	if err := h.queries.ClearTaskTags(r.Context(), db.ClearTaskTagsParams{TaskID: id, UserID: userID}); err != nil {
		writeDBError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /users/{user_id}/tags/{tag_id}/tasks
func (h *taskHandler) listByTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	tagID, ok := readPathID(w, r, "tag_id")
	if !ok {
		return
	}

	rows, err := h.queries.ListTasksByTag(r.Context(), db.ListTasksByTagParams{TagID: tagID, UserID: userID})
	if err != nil {
		writeDBError(w, err)
		return
	}
	tasks := make([]db.Task, 0, len(rows))
	for _, t := range rows {
		tasks = append(tasks, db.Task(t))
	}
	writeTasks(w, tasks)
}

// taskTags devuelve los tags de una tarea ya convertidos a su respuesta JSON.
func (h *taskHandler) taskTags(r *http.Request, taskID, userID int32) ([]tagResponse, error) {
	rows, err := h.queries.ListTagsByTask(r.Context(), db.ListTagsByTaskParams{TaskID: taskID, UserID: userID})
	if err != nil {
		return nil, err
	}
	out := make([]tagResponse, 0, len(rows))
	for _, t := range rows {
		out = append(out, newTagResponse(db.Tag(t)))
	}
	return out, nil
}
