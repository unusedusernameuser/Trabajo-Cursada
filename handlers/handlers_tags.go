package main

import (
	"net/http"
	"time"

	db "my_sqlc_project/db/sqlc"
)

// tagHandler agrupa los handlers CRUD de la entidad tags.
// Los tags pertenecen a un usuario, por eso cuelgan de /users/{user_id}.
// (Los endpoints para asociar tags a tareas están en handlers_tasks.go.)
type tagHandler struct {
	queries *db.Queries
}

type tagInput struct {
	TagName string `json:"tag_name"`
}

type tagResponse struct {
	ID        int32      `json:"id"`
	UserID    int32      `json:"user_id"`
	TagName   string     `json:"tag_name"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func newTagResponse(t db.Tag) tagResponse {
	return tagResponse{
		ID:        t.ID,
		UserID:    t.UserID,
		TagName:   t.TagName,
		CreatedAt: nullTime(t.CreatedAt),
		UpdatedAt: nullTime(t.UpdatedAt),
	}
}

func registerTagRoutes(queries *db.Queries) {
	h := &tagHandler{queries: queries}

	http.HandleFunc("POST /users/{user_id}/tags", h.create)
	http.HandleFunc("GET /users/{user_id}/tags", h.list)
	http.HandleFunc("GET /users/{user_id}/tags/{id}", h.get)
	http.HandleFunc("PUT /users/{user_id}/tags/{id}", h.update)
	http.HandleFunc("DELETE /users/{user_id}/tags/{id}", h.delete)
}

func readTagInput(w http.ResponseWriter, r *http.Request) (tagInput, bool) {
	var in tagInput
	if !decodeBody(w, r, &in) || !checkText(w, "tag_name", in.TagName, 63) {
		return in, false
	}
	return in, true
}

// POST /users/{user_id}/tags
func (h *tagHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	in, ok := readTagInput(w, r)
	if !ok {
		return
	}

	tag, err := h.queries.CreateTag(r.Context(), db.CreateTagParams{UserID: userID, TagName: in.TagName})
	if err != nil {
		writeDBErrorMsg(w, err, "usuario no encontrado", "ya existe un tag con ese nombre")
		return
	}

	writeJSON(w, http.StatusCreated, newTagResponse(tag))
}

// GET /users/{user_id}/tags
func (h *tagHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}

	tags, err := h.queries.ListTagsByUser(r.Context(), userID)
	if err != nil {
		writeDBError(w, err)
		return
	}

	out := make([]tagResponse, 0, len(tags))
	for _, t := range tags {
		out = append(out, newTagResponse(t))
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /users/{user_id}/tags/{id}
func (h *tagHandler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}

	tag, err := h.queries.GetTagByID(r.Context(), db.GetTagByIDParams{ID: id, UserID: userID})
	if err != nil {
		writeDBErrorMsg(w, err, "tag no encontrado", "")
		return
	}

	writeJSON(w, http.StatusOK, newTagResponse(tag))
}

// PUT /users/{user_id}/tags/{id}
func (h *tagHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	in, ok := readTagInput(w, r)
	if !ok {
		return
	}

	tag, err := h.queries.UpdateTagName(r.Context(), db.UpdateTagNameParams{
		ID:      id,
		UserID:  userID,
		TagName: in.TagName,
	})
	if err != nil {
		writeDBErrorMsg(w, err, "tag no encontrado", "ya existe un tag con ese nombre")
		return
	}

	writeJSON(w, http.StatusOK, newTagResponse(tag))
}

// DELETE /users/{user_id}/tags/{id}
func (h *tagHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}

	if err := h.queries.DeleteTag(r.Context(), db.DeleteTagParams{ID: id, UserID: userID}); err != nil {
		writeDBError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
