package main

import (
	"net/http"
	"time"

	db "my_sqlc_project/db/sqlc"
)

// folderHandler agrupa los handlers CRUD de la entidad folders.
// Las carpetas pertenecen a un usuario, por eso cuelgan de /users/{user_id}.
type folderHandler struct {
	queries *db.Queries
}

type folderInput struct {
	FolderName string `json:"folder_name"`
}

type folderResponse struct {
	ID         int32      `json:"id"`
	UserID     int32      `json:"user_id"`
	FolderName string     `json:"folder_name"`
	CreatedAt  *time.Time `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

func newFolderResponse(f db.Folder) folderResponse {
	return folderResponse{
		ID:         f.ID,
		UserID:     f.UserID,
		FolderName: f.FolderName,
		CreatedAt:  nullTime(f.CreatedAt),
		UpdatedAt:  nullTime(f.UpdatedAt),
	}
}

func registerFolderRoutes(queries *db.Queries) {
	h := &folderHandler{queries: queries}

	http.HandleFunc("POST /users/{user_id}/folders", h.create)
	http.HandleFunc("GET /users/{user_id}/folders", h.list)
	http.HandleFunc("GET /users/{user_id}/folders/{id}", h.get)
	http.HandleFunc("PUT /users/{user_id}/folders/{id}", h.update)
	http.HandleFunc("DELETE /users/{user_id}/folders/{id}", h.delete)
}

func readFolderInput(w http.ResponseWriter, r *http.Request) (folderInput, bool) {
	var in folderInput
	if !decodeBody(w, r, &in) || !checkText(w, "folder_name", in.FolderName, 63) {
		return in, false
	}
	return in, true
}

// POST /users/{user_id}/folders
func (h *folderHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	in, ok := readFolderInput(w, r)
	if !ok {
		return
	}

	folder, err := h.queries.CreateFolder(r.Context(), db.CreateFolderParams{
		UserID:     userID,
		FolderName: in.FolderName,
	})
	if err != nil {
		writeDBErrorMsg(w, err, "usuario no encontrado", "ya existe una carpeta con ese nombre")
		return
	}

	writeJSON(w, http.StatusCreated, newFolderResponse(folder))
}

// GET /users/{user_id}/folders
func (h *folderHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}

	folders, err := h.queries.ListFoldersByUser(r.Context(), userID)
	if err != nil {
		writeDBError(w, err)
		return
	}

	out := make([]folderResponse, 0, len(folders)) // para responder [] y no null
	for _, f := range folders {
		out = append(out, newFolderResponse(f))
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /users/{user_id}/folders/{id}
func (h *folderHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}

	folder, err := h.queries.GetFolderByID(r.Context(), db.GetFolderByIDParams{ID: id, UserID: userID})
	if err != nil {
		writeDBErrorMsg(w, err, "carpeta no encontrada", "")
		return
	}

	writeJSON(w, http.StatusOK, newFolderResponse(folder))
}

// PUT /users/{user_id}/folders/{id}
func (h *folderHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	in, ok := readFolderInput(w, r)
	if !ok {
		return
	}

	folder, err := h.queries.UpdateFolderName(r.Context(), db.UpdateFolderNameParams{
		ID:         id,
		UserID:     userID,
		FolderName: in.FolderName,
	})
	if err != nil {
		writeDBErrorMsg(w, err, "carpeta no encontrada", "ya existe una carpeta con ese nombre")
		return
	}

	writeJSON(w, http.StatusOK, newFolderResponse(folder))
}

// DELETE /users/{user_id}/folders/{id}
// Las tareas de la carpeta no se borran: quedan sin carpeta (ON DELETE SET NULL).
func (h *folderHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}

	if err := h.queries.DeleteFolder(r.Context(), db.DeleteFolderParams{ID: id, UserID: userID}); err != nil {
		writeDBError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
