package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	db "my_sqlc_project/db/sqlc"
	"my_sqlc_project/views"
)

// tasksHandler agrupa los handlers de la página de tareas de un usuario: tareas (este archivo),
// carpetas (folders.go) y tags (tags.go). Todas las rutas cuelgan de /users/{user_id} y todas
// las queries filtran por ese usuario: nunca se ve ni se modifica lo de otro usuario.
type tasksHandler struct {
	queries *db.Queries
}

// ---- Página -------------------------------------------------------------------

// GET /users/{user_id}/tasks[?folder=&tag=&status=&priority=]
// Los filtros viajan en la URL; la lista se filtra en el servidor.
func (h *tasksHandler) index(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	h.render(w, r, userID, views.FiltersFromValues(r.URL.Query()), http.StatusOK, "", nil)
}

// render arma los datos de la página y escribe TasksPage. Se usa también para volver a mostrar
// la página con un mensaje de error (400/404/409). Si form no es nil, el formulario de alta
// se muestra con esos valores.
func (h *tasksHandler) render(w http.ResponseWriter, r *http.Request, userID int32, filters views.TaskFilters, status int, errMsg string, form *views.TaskFormValues) {
	data, err := h.buildPage(r.Context(), userID, filters)
	if err != nil {
		if isNoRows(err) { // el usuario no existe
			notFound(w, r)
			return
		}
		serverError(w, "error al armar la página de tareas", err)
		return
	}
	data.Error = errMsg
	if form != nil {
		data.NewTask = *form
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := views.TasksPage(data).Render(r.Context(), w); err != nil {
		log.Printf("error al renderizar la página de tareas: %v", err)
	}
}

// buildPage lee de la base todo lo que muestra la página. Devuelve sql.ErrNoRows si el usuario no existe.
func (h *tasksHandler) buildPage(ctx context.Context, userID int32, filters views.TaskFilters) (views.TasksPageData, error) {
	user, err := h.queries.GetUserByID(ctx, userID)
	if err != nil {
		return views.TasksPageData{}, err
	}
	tasks, err := h.queries.ListTasksByUser(ctx, db.ListTasksByUserParams{UserID: userID})
	if err != nil {
		return views.TasksPageData{}, err
	}
	folders, err := h.queries.ListFoldersByUser(ctx, userID)
	if err != nil {
		return views.TasksPageData{}, err
	}
	tags, err := h.queries.ListTagsByUser(ctx, userID)
	if err != nil {
		return views.TasksPageData{}, err
	}

	// Los tags de cada tarea se arman con una consulta por tag (ListTasksByTag).
	// Como los tags vienen ordenados por nombre, cada tarea también los tiene ordenados.
	tagsOfTask := map[int32][]db.Tag{}
	tagCount := map[int32]int{}
	for _, tag := range tags {
		rows, err := h.queries.ListTasksByTag(ctx, db.ListTasksByTagParams{TagID: tag.ID, UserID: userID})
		if err != nil {
			return views.TasksPageData{}, err
		}
		tagCount[tag.ID] = len(rows)
		for _, row := range rows {
			tagsOfTask[row.ID] = append(tagsOfTask[row.ID], tag)
		}
	}

	// Si el filtro apunta a una carpeta o un tag que ya no existe, se vuelve a "todas".
	if filters.Folder != "all" && filters.Folder != "none" &&
		!slices.ContainsFunc(folders, func(f db.Folder) bool { return idText(f.ID) == filters.Folder }) {
		filters.Folder = "all"
	}
	if filters.Tag != 0 && !slices.ContainsFunc(tags, func(t db.Tag) bool { return t.ID == filters.Tag }) {
		filters.Tag = 0
	}

	// Carpetas con la cantidad de tareas de cada una.
	folderCount := map[int32]int{}
	withoutFolder := 0
	for _, t := range tasks {
		if t.FolderID.Valid {
			folderCount[t.FolderID.Int32]++
		} else {
			withoutFolder++
		}
	}
	folderRefs := make([]views.NamedRef, 0, len(folders))
	for _, f := range folders {
		folderRefs = append(folderRefs, views.NamedRef{ID: f.ID, Name: f.FolderName, Count: folderCount[f.ID]})
	}
	tagRefs := make([]views.NamedRef, 0, len(tags))
	for _, t := range tags {
		tagRefs = append(tagRefs, views.NamedRef{ID: t.ID, Name: t.TagName, Count: tagCount[t.ID]})
	}

	// Tareas visibles: las completadas van al final; dentro de cada bloque se mantiene
	// el orden de la query (por fecha límite).
	now := time.Now()
	var pending, done []views.TaskRow
	for _, t := range tasks {
		assigned := tagsOfTask[t.ID]
		if !matchesFilters(t, assigned, filters) {
			continue
		}
		row := views.TaskRow{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.TaskDescription.String,
			Status:      t.TaskStatus,
			Priority:    t.TaskPriority,
			DueText:     views.FormatDateTime(t.DueDate),
			Overdue:     t.DueDate.Valid && t.TaskStatus != "completada" && t.DueDate.Time.Before(now),
			FolderID:    t.FolderID.Int32, // 0 si es NULL
		}
		for _, tag := range tags {
			ref := views.NamedRef{ID: tag.ID, Name: tag.TagName}
			if slices.ContainsFunc(assigned, func(a db.Tag) bool { return a.ID == tag.ID }) {
				row.Tags = append(row.Tags, ref)
			} else {
				row.FreeTags = append(row.FreeTags, ref)
			}
		}
		if t.TaskStatus == "completada" {
			done = append(done, row)
		} else {
			pending = append(pending, row)
		}
	}
	visible := append(pending, done...)

	emptyText := "Todavía no hay tareas. Creá la primera con el formulario."
	if len(tasks) > 0 && len(visible) == 0 {
		emptyText = "Ninguna tarea coincide con los filtros elegidos."
	}

	// El formulario de alta propone la carpeta que se está filtrando.
	newTask := views.TaskFormValues{Priority: "media"}
	if id, ok := parseID(filters.Folder); ok {
		newTask.FolderID = id
	}

	return views.TasksPageData{
		UserID:        user.ID,
		DisplayName:   user.DisplayName,
		Handle:        user.Handle,
		Email:         user.Email,
		Filters:       filters,
		FilterSummary: filterSummary(filters, folders, tags),
		EmptyText:     emptyText,
		TotalTasks:    len(tasks),
		WithoutFolder: withoutFolder,
		Folders:       folderRefs,
		Tags:          tagRefs,
		Tasks:         visible,
		NewTask:       newTask,
	}, nil
}

// idText es el id como texto, tal como viaja en los filtros de la URL.
func idText(id int32) string {
	return strconv.Itoa(int(id))
}

func matchesFilters(t db.Task, taskTags []db.Tag, f views.TaskFilters) bool {
	switch {
	case f.Folder == "none" && t.FolderID.Valid:
		return false
	case f.Folder != "all" && f.Folder != "none" && (!t.FolderID.Valid || idText(t.FolderID.Int32) != f.Folder):
		return false
	case f.Tag != 0 && !slices.ContainsFunc(taskTags, func(tag db.Tag) bool { return tag.ID == f.Tag }):
		return false
	case f.Status != "" && t.TaskStatus != f.Status:
		return false
	case f.Priority != "" && t.TaskPriority != f.Priority:
		return false
	}
	return true
}

func filterSummary(f views.TaskFilters, folders []db.Folder, tags []db.Tag) string {
	var parts []string
	switch f.Folder {
	case "all":
	case "none":
		parts = append(parts, "sin carpeta")
	default:
		for _, folder := range folders {
			if idText(folder.ID) == f.Folder {
				parts = append(parts, "carpeta «"+folder.FolderName+"»")
			}
		}
	}
	if f.Tag != 0 {
		for _, tag := range tags {
			if tag.ID == f.Tag {
				parts = append(parts, "tag «"+tag.TagName+"»")
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Filtrando por " + strings.Join(parts, " y ") + ". Volvé a hacer clic en el filtro para quitarlo."
}

// ---- Alta, edición y baja de tareas ---------------------------------------------

// taskFields lee y valida los campos que comparten el alta y la edición.
// Devuelve el mensaje de error ("" si está todo bien) y la fecha ya interpretada.
func taskFields(title, priority, due string) (sql.NullTime, string) {
	if msg := checkText("el título", title, 127); msg != "" {
		return sql.NullTime{}, msg
	}
	if !slices.Contains(views.Priorities, priority) {
		return sql.NullTime{}, "la prioridad debe ser baja, media o alta"
	}
	dueDate, ok := parseDateTime(due)
	if !ok {
		return sql.NullTime{}, "la fecha límite no es válida"
	}
	return dueDate, ""
}

// POST /users/{user_id}/tasks
// Crea la tarea (con sus tags) y redirige a la lista (PRG). Si algo es inválido vuelve a
// mostrar la página con el error y lo que se escribió.
func (h *tasksHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	filters := backFilters(r)

	form := views.TaskFormValues{
		Title:       strings.TrimSpace(r.PostFormValue("title")),
		Description: strings.TrimSpace(r.PostFormValue("task_description")),
		Due:         strings.TrimSpace(r.PostFormValue("due_date")),
		Priority:    r.PostFormValue("task_priority"),
	}
	if form.Priority == "" {
		form.Priority = "media"
	}
	folderID, folderOK := parseOptionalID(r.PostFormValue("folder_id"))
	form.FolderID = folderID

	tagsOK := true
	for _, raw := range r.PostForm["tag_ids"] {
		id, ok := parseID(raw)
		if !ok {
			tagsOK = false
			break
		}
		form.TagIDs = append(form.TagIDs, id)
	}

	dueDate, msg := taskFields(form.Title, form.Priority, form.Due)
	switch {
	case msg != "":
	case !folderOK:
		msg = "la carpeta no es válida"
	case !tagsOK:
		msg = "alguno de los tags no es válido"
	}
	if msg != "" {
		h.render(w, r, userID, filters, http.StatusBadRequest, msg, &form)
		return
	}

	ctx := r.Context()
	task, err := h.queries.CreateTask(ctx, db.CreateTaskParams{
		UserID:          userID,
		FolderID:        nullID(form.FolderID),
		Title:           form.Title,
		TaskDescription: nullText(form.Description),
		DueDate:         dueDate,
		TaskPriority:    sql.NullString{String: form.Priority, Valid: true},
	})
	if err != nil {
		switch {
		case isNoRows(err): // la query no inserta si la carpeta no es del usuario
			h.render(w, r, userID, filters, http.StatusBadRequest, "la carpeta debe ser del usuario", &form)
		case pgCode(err) == pgForeignKeyViolation: // el usuario no existe
			notFound(w, r)
		default:
			serverError(w, "error al crear la tarea", err)
		}
		return
	}

	// Los tags que no son del usuario simplemente no se asocian (la query no inserta nada).
	for _, tagID := range form.TagIDs {
		if err := h.queries.AddTagToTask(ctx, db.AddTagToTaskParams{TaskID: task.ID, TagID: tagID, UserID: userID}); err != nil {
			serverError(w, "error al asociar un tag a la tarea", err)
			return
		}
	}

	redirectToTasks(w, r, userID)
}

// GET /users/{user_id}/tasks/{id}/edit[?folder=&tag=&status=&priority=]
// Página con el formulario de edición. Los filtros se conservan para volver a la misma lista.
func (h *tasksHandler) editForm(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	task, err := h.queries.GetTaskByID(r.Context(), db.GetTaskByIDParams{ID: taskID, UserID: userID})
	if err != nil && !isNoRows(err) {
		serverError(w, "error al leer la tarea", err)
		return
	}
	// Solo el creador puede editarla (UpdateTask filtra igual por user_id).
	if err != nil || task.UserID != userID {
		notFound(w, r)
		return
	}

	renderEdit(w, r, views.TaskEditData{
		UserID:      userID,
		TaskID:      taskID,
		Title:       task.Title,
		Description: task.TaskDescription.String,
		Due:         views.InputDateTime(task.DueDate),
		Priority:    task.TaskPriority,
		Filters:     views.FiltersFromValues(r.URL.Query()),
	}, http.StatusOK)
}

func renderEdit(w http.ResponseWriter, r *http.Request, data views.TaskEditData, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := views.TaskEditPage(data).Render(r.Context(), w); err != nil {
		log.Printf("error al renderizar la edición de la tarea: %v", err)
	}
}

// POST /users/{user_id}/tasks/{id}
// Modifica título, descripción, fecha límite y prioridad; redirige a la lista (PRG).
func (h *tasksHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	filters := backFilters(r)

	data := views.TaskEditData{
		UserID:      userID,
		TaskID:      taskID,
		Title:       strings.TrimSpace(r.PostFormValue("title")),
		Description: strings.TrimSpace(r.PostFormValue("task_description")),
		Due:         strings.TrimSpace(r.PostFormValue("due_date")),
		Priority:    r.PostFormValue("task_priority"),
		Filters:     filters,
	}
	dueDate, msg := taskFields(data.Title, data.Priority, data.Due)
	if msg != "" {
		data.Error = msg
		renderEdit(w, r, data, http.StatusBadRequest)
		return
	}

	_, err := h.queries.UpdateTask(r.Context(), db.UpdateTaskParams{
		ID:              taskID,
		UserID:          userID,
		Title:           data.Title,
		TaskDescription: nullText(data.Description),
		DueDate:         dueDate,
		TaskPriority:    data.Priority,
	})
	if err != nil {
		if isNoRows(err) {
			notFound(w, r)
			return
		}
		serverError(w, "error al actualizar la tarea", err)
		return
	}

	redirectToTasks(w, r, userID)
}

// POST /users/{user_id}/tasks/{id}/delete
// Los formularios HTML solo soportan GET y POST, por eso el borrado es un POST.
func (h *tasksHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}

	if err := h.queries.DeleteTask(r.Context(), db.DeleteTaskParams{ID: taskID, UserID: userID}); err != nil {
		serverError(w, "error al eliminar la tarea", err)
		return
	}
	redirectToTasks(w, r, userID)
}

// ---- Cambios parciales ------------------------------------------------------------

// POST /users/{user_id}/tasks/{id}/status   (task_status=pendiente|en_progreso|completada)
// completed_at se completa al pasar a "completada" y vuelve a NULL con cualquier otro estado.
func (h *tasksHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	status := r.PostFormValue("task_status")
	if !slices.Contains(views.Statuses, status) {
		h.render(w, r, userID, backFilters(r), http.StatusBadRequest, "el estado debe ser pendiente, en_progreso o completada", nil)
		return
	}

	_, err := h.queries.UpdateTaskStatus(r.Context(), db.UpdateTaskStatusParams{TaskStatus: status, ID: taskID, UserID: userID})
	if err != nil {
		if isNoRows(err) {
			h.render(w, r, userID, backFilters(r), http.StatusNotFound, "tarea no encontrada", nil)
			return
		}
		serverError(w, "error al cambiar el estado de la tarea", err)
		return
	}
	redirectToTasks(w, r, userID)
}

// POST /users/{user_id}/tasks/{id}/folder   (folder_id=3, o vacío para sacarla de la carpeta)
// La carpeta destino debe ser del usuario.
func (h *tasksHandler) moveToFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	folderID, ok := parseOptionalID(r.PostFormValue("folder_id"))
	if !ok {
		h.render(w, r, userID, backFilters(r), http.StatusBadRequest, "la carpeta no es válida", nil)
		return
	}

	_, err := h.queries.MoveTaskToFolder(r.Context(), db.MoveTaskToFolderParams{FolderID: nullID(folderID), ID: taskID, UserID: userID})
	if err != nil {
		if isNoRows(err) {
			h.render(w, r, userID, backFilters(r), http.StatusNotFound, "tarea no encontrada o la carpeta no es del usuario", nil)
			return
		}
		serverError(w, "error al mover la tarea", err)
		return
	}
	redirectToTasks(w, r, userID)
}

// ---- Tags de una tarea ----------------------------------------------------------------

// POST /users/{user_id}/tasks/{id}/tags   (tag_id=2)
// Asocia el tag a la tarea (idempotente). Tarea y tag deben ser del usuario.
func (h *tasksHandler) addTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	tagID, ok := parseID(r.PostFormValue("tag_id"))
	if !ok {
		h.render(w, r, userID, backFilters(r), http.StatusBadRequest, "elegí un tag", nil)
		return
	}

	ctx := r.Context()
	// La query no falla si la tarea o el tag no son del usuario: simplemente no inserta.
	// Por eso se verifica el resultado para poder avisar.
	if err := h.queries.AddTagToTask(ctx, db.AddTagToTaskParams{TaskID: taskID, TagID: tagID, UserID: userID}); err != nil {
		serverError(w, "error al agregar el tag a la tarea", err)
		return
	}
	tags, err := h.queries.ListTagsByTask(ctx, db.ListTagsByTaskParams{TaskID: taskID, UserID: userID})
	if err != nil {
		serverError(w, "error al leer los tags de la tarea", err)
		return
	}
	found := false
	for _, t := range tags {
		if t.ID == tagID {
			found = true
			break
		}
	}
	if !found {
		h.render(w, r, userID, backFilters(r), http.StatusNotFound, "la tarea o el tag no existen o no son del usuario", nil)
		return
	}
	redirectToTasks(w, r, userID)
}

// POST /users/{user_id}/tasks/{id}/tags/{tag_id}/delete
func (h *tasksHandler) removeTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	taskID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	tagID, ok := pathID(w, r, "tag_id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}

	if err := h.queries.RemoveTagFromTask(r.Context(), db.RemoveTagFromTaskParams{TaskID: taskID, TagID: tagID, UserID: userID}); err != nil {
		serverError(w, "error al quitar el tag de la tarea", err)
		return
	}
	redirectToTasks(w, r, userID)
}
