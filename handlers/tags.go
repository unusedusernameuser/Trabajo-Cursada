package handlers

import (
	"net/http"
	"strings"

	db "my_sqlc_project/db/sqlc"
	"my_sqlc_project/views"
)

// POST /users/{user_id}/tags   (tag_name)
// (Asociar y quitar tags de una tarea está en tasks.go.)
func (h *tasksHandler) createTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	filters := backFilters(r)

	name := strings.TrimSpace(r.PostFormValue("tag_name"))
	if msg := checkText("el nombre del tag", name, 63); msg != "" {
		h.render(w, r, userID, filters, http.StatusBadRequest, msg, nil)
		return
	}

	if _, err := h.queries.CreateTag(r.Context(), db.CreateTagParams{UserID: userID, TagName: name}); err != nil {
		h.tagError(w, r, userID, filters, err, "error al crear el tag")
		return
	}
	redirectToTasks(w, r, userID)
}

// POST /users/{user_id}/tags/{id}   (tag_name)
func (h *tasksHandler) renameTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	tagID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	filters := backFilters(r)

	name := strings.TrimSpace(r.PostFormValue("tag_name"))
	if msg := checkText("el nombre del tag", name, 63); msg != "" {
		h.render(w, r, userID, filters, http.StatusBadRequest, msg, nil)
		return
	}

	_, err := h.queries.UpdateTagName(r.Context(), db.UpdateTagNameParams{TagName: name, ID: tagID, UserID: userID})
	if err != nil {
		h.tagError(w, r, userID, filters, err, "error al renombrar el tag")
		return
	}
	redirectToTasks(w, r, userID)
}

// POST /users/{user_id}/tags/{id}/delete
// El tag se quita de todas las tareas que lo tengan (ON DELETE CASCADE en tasks_tags).
func (h *tasksHandler) deleteTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	tagID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}

	if err := h.queries.DeleteTag(r.Context(), db.DeleteTagParams{ID: tagID, UserID: userID}); err != nil {
		serverError(w, "error al eliminar el tag", err)
		return
	}
	redirectToTasks(w, r, userID)
}

// tagError traduce un error de la base al estado HTTP y vuelve a mostrar la página.
func (h *tasksHandler) tagError(w http.ResponseWriter, r *http.Request, userID int32, filters views.TaskFilters, err error, what string) {
	switch {
	case isNoRows(err): // renombrar: el tag no existe o no es del usuario
		h.render(w, r, userID, filters, http.StatusNotFound, "tag no encontrado", nil)
	case pgCode(err) == pgUniqueViolation:
		h.render(w, r, userID, filters, http.StatusConflict, "ya existe un tag con ese nombre", nil)
	case pgCode(err) == pgForeignKeyViolation: // crear: el usuario no existe
		notFound(w, r)
	default:
		serverError(w, what, err)
	}
}
