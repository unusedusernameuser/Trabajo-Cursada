package handlers

import (
	"net/http"
	"strings"

	db "my_sqlc_project/db/sqlc"
	"my_sqlc_project/views"
)

// POST /users/{user_id}/folders   (folder_name)
func (h *tasksHandler) createFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	filters := backFilters(r)

	name := strings.TrimSpace(r.PostFormValue("folder_name"))
	if msg := checkText("el nombre de la carpeta", name, 63); msg != "" {
		h.render(w, r, userID, filters, http.StatusBadRequest, msg, nil)
		return
	}

	if _, err := h.queries.CreateFolder(r.Context(), db.CreateFolderParams{UserID: userID, FolderName: name}); err != nil {
		h.folderError(w, r, userID, filters, err, "error al crear la carpeta")
		return
	}
	redirectToTasks(w, r, userID)
}

// POST /users/{user_id}/folders/{id}   (folder_name)
// Los formularios HTML solo soportan GET y POST, por eso renombrar es un POST.
func (h *tasksHandler) renameFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	folderID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	filters := backFilters(r)

	name := strings.TrimSpace(r.PostFormValue("folder_name"))
	if msg := checkText("el nombre de la carpeta", name, 63); msg != "" {
		h.render(w, r, userID, filters, http.StatusBadRequest, msg, nil)
		return
	}

	_, err := h.queries.UpdateFolderName(r.Context(), db.UpdateFolderNameParams{FolderName: name, ID: folderID, UserID: userID})
	if err != nil {
		h.folderError(w, r, userID, filters, err, "error al renombrar la carpeta")
		return
	}
	redirectToTasks(w, r, userID)
}

// POST /users/{user_id}/folders/{id}/delete
// Las tareas de la carpeta no se borran: quedan sin carpeta (ON DELETE SET NULL).
func (h *tasksHandler) deleteFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "user_id")
	if !ok {
		return
	}
	folderID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}

	if err := h.queries.DeleteFolder(r.Context(), db.DeleteFolderParams{ID: folderID, UserID: userID}); err != nil {
		serverError(w, "error al eliminar la carpeta", err)
		return
	}
	redirectToTasks(w, r, userID)
}

// folderError traduce un error de la base al estado HTTP y vuelve a mostrar la página.
func (h *tasksHandler) folderError(w http.ResponseWriter, r *http.Request, userID int32, filters views.TaskFilters, err error, what string) {
	switch {
	case isNoRows(err): // renombrar: la carpeta no existe o no es del usuario
		h.render(w, r, userID, filters, http.StatusNotFound, "carpeta no encontrada", nil)
	case pgCode(err) == pgUniqueViolation:
		h.render(w, r, userID, filters, http.StatusConflict, "ya existe una carpeta con ese nombre", nil)
	case pgCode(err) == pgForeignKeyViolation: // crear: el usuario no existe
		notFound(w, r)
	default:
		serverError(w, what, err)
	}
}
