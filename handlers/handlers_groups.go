package main

import (
	"net/http"
	"time"

	db "my_sqlc_project/db/sqlc"
)

// groupHandler agrupa los handlers de groups y groups_members.
// Crear, modificar y borrar un grupo requiere ser su dueño, por eso esas rutas
// cuelgan de /users/{user_id}/groups. La consulta y los miembros cuelgan de /groups/{id}.
type groupHandler struct {
	queries *db.Queries
}

type groupInput struct {
	GroupName        string  `json:"group_name"`
	GroupDescription *string `json:"group_description"`
}

type groupResponse struct {
	ID               int32      `json:"id"`
	OwnerID          int32      `json:"owner_id"`
	GroupName        string     `json:"group_name"`
	GroupDescription *string    `json:"group_description"`
	CreatedAt        *time.Time `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

func newGroupResponse(g db.Group) groupResponse {
	return groupResponse{
		ID:               g.ID,
		OwnerID:          g.OwnerID,
		GroupName:        g.GroupName,
		GroupDescription: nullString(g.GroupDescription),
		CreatedAt:        nullTime(g.CreatedAt),
		UpdatedAt:        nullTime(g.UpdatedAt),
	}
}

type memberInput struct {
	UserID int32 `json:"user_id"`
}

type memberResponse struct {
	GroupID  int32      `json:"group_id"`
	UserID   int32      `json:"user_id"`
	JoinedAt *time.Time `json:"joined_at"`
}

type memberUserResponse struct {
	ID          int32      `json:"id"`
	Handle      string     `json:"handle"`
	DisplayName string     `json:"display_name"`
	Email       string     `json:"email"`
	JoinedAt    *time.Time `json:"joined_at"`
}

func registerGroupRoutes(queries *db.Queries) {
	h := &groupHandler{queries: queries}

	// Grupos
	http.HandleFunc("POST /users/{user_id}/groups", h.create)
	http.HandleFunc("GET /users/{user_id}/groups", h.list) // ?owner=true: solo los que creó
	http.HandleFunc("GET /groups/{id}", h.get)
	http.HandleFunc("PUT /users/{user_id}/groups/{id}", h.update)
	http.HandleFunc("DELETE /users/{user_id}/groups/{id}", h.delete)

	// Miembros
	http.HandleFunc("POST /groups/{id}/members", h.addMember)
	http.HandleFunc("GET /groups/{id}/members", h.listMembers)
	http.HandleFunc("GET /groups/{id}/members/count", h.countMembers)
	http.HandleFunc("GET /groups/{id}/members/{user_id}", h.isMember)
	http.HandleFunc("DELETE /groups/{id}/members/{user_id}", h.removeMember)
}

func readGroupInput(w http.ResponseWriter, r *http.Request) (groupInput, bool) {
	var in groupInput
	if !decodeBody(w, r, &in) || !checkText(w, "group_name", in.GroupName, 63) {
		return in, false
	}
	return in, true
}

// POST /users/{user_id}/groups
// El usuario queda como dueño y se agrega automáticamente como miembro.
func (h *groupHandler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	in, ok := readGroupInput(w, r)
	if !ok {
		return
	}

	group, err := h.queries.CreateGroup(r.Context(), db.CreateGroupParams{
		OwnerID:          userID,
		GroupName:        in.GroupName,
		GroupDescription: toNullString(in.GroupDescription),
	})
	if err != nil {
		writeDBErrorMsg(w, err, "usuario no encontrado", "ya existe un grupo con esos datos")
		return
	}

	writeJSON(w, http.StatusCreated, newGroupResponse(db.Group(group)))
}

// GET /users/{user_id}/groups[?owner=true]
func (h *groupHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}

	var groups []db.Group
	if r.URL.Query().Get("owner") == "true" {
		rows, err := h.queries.ListGroupsByOwner(r.Context(), userID)
		if err != nil {
			writeDBError(w, err)
			return
		}
		for _, g := range rows {
			groups = append(groups, db.Group(g))
		}
	} else {
		rows, err := h.queries.ListGroupsByMember(r.Context(), userID)
		if err != nil {
			writeDBError(w, err)
			return
		}
		for _, g := range rows {
			groups = append(groups, db.Group(g))
		}
	}

	out := make([]groupResponse, 0, len(groups))
	for _, g := range groups {
		out = append(out, newGroupResponse(g))
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /groups/{id}
func (h *groupHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}

	group, err := h.queries.GetGroupByID(r.Context(), id)
	if err != nil {
		writeDBErrorMsg(w, err, "grupo no encontrado", "")
		return
	}

	writeJSON(w, http.StatusOK, newGroupResponse(db.Group(group)))
}

// PUT /users/{user_id}/groups/{id}
// Solo el dueño puede modificarlo; si no lo es la query no encuentra el grupo (404).
func (h *groupHandler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}
	in, ok := readGroupInput(w, r)
	if !ok {
		return
	}

	group, err := h.queries.UpdateGroup(r.Context(), db.UpdateGroupParams{
		ID:               id,
		OwnerID:          userID,
		GroupName:        in.GroupName,
		GroupDescription: toNullString(in.GroupDescription),
	})
	if err != nil {
		writeDBErrorMsg(w, err, "grupo no encontrado o el usuario no es su dueño", "")
		return
	}

	writeJSON(w, http.StatusOK, newGroupResponse(db.Group(group)))
}

// DELETE /users/{user_id}/groups/{id}
// Solo el dueño puede borrarlo. También se borran sus tareas y membresías (ON DELETE CASCADE).
func (h *groupHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}
	id, ok := readID(w, r)
	if !ok {
		return
	}

	if err := h.queries.DeleteGroup(r.Context(), db.DeleteGroupParams{ID: id, OwnerID: userID}); err != nil {
		writeDBError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /groups/{id}/members   {"user_id": 3}
func (h *groupHandler) addMember(w http.ResponseWriter, r *http.Request) {
	groupID, ok := readID(w, r)
	if !ok {
		return
	}
	var in memberInput
	if !decodeBody(w, r, &in) {
		return
	}
	if in.UserID <= 0 {
		writeError(w, http.StatusBadRequest, "user_id debe ser un entero positivo")
		return
	}

	m, err := h.queries.AddGroupMember(r.Context(), db.AddGroupMemberParams{GroupID: groupID, UserID: in.UserID})
	if err != nil {
		writeDBErrorMsg(w, err, "grupo o usuario no encontrado", "el usuario ya es miembro del grupo")
		return
	}

	writeJSON(w, http.StatusCreated, memberResponse{
		GroupID:  m.GroupID,
		UserID:   m.UserID,
		JoinedAt: nullTime(m.JoinedAt),
	})
}

// GET /groups/{id}/members
func (h *groupHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	groupID, ok := readID(w, r)
	if !ok {
		return
	}

	rows, err := h.queries.ListGroupMembers(r.Context(), groupID)
	if err != nil {
		writeDBError(w, err)
		return
	}

	out := make([]memberUserResponse, 0, len(rows))
	for _, m := range rows {
		out = append(out, memberUserResponse{
			ID:          m.ID,
			Handle:      m.Handle,
			DisplayName: m.DisplayName,
			Email:       m.Email,
			JoinedAt:    nullTime(m.JoinedAt),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /groups/{id}/members/count
func (h *groupHandler) countMembers(w http.ResponseWriter, r *http.Request) {
	groupID, ok := readID(w, r)
	if !ok {
		return
	}

	count, err := h.queries.CountGroupMembers(r.Context(), groupID)
	if err != nil {
		writeDBError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"count": count})
}

// GET /groups/{id}/members/{user_id}
func (h *groupHandler) isMember(w http.ResponseWriter, r *http.Request) {
	groupID, ok := readID(w, r)
	if !ok {
		return
	}
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}

	isMember, err := h.queries.IsGroupMember(r.Context(), db.IsGroupMemberParams{GroupID: groupID, UserID: userID})
	if err != nil {
		writeDBError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"is_member": isMember})
}

// DELETE /groups/{id}/members/{user_id}
// La query no permite quitar al dueño del grupo (no falla, pero no borra nada).
func (h *groupHandler) removeMember(w http.ResponseWriter, r *http.Request) {
	groupID, ok := readID(w, r)
	if !ok {
		return
	}
	userID, ok := readPathID(w, r, "user_id")
	if !ok {
		return
	}

	if err := h.queries.RemoveGroupMember(r.Context(), db.RemoveGroupMemberParams{GroupID: groupID, UserID: userID}); err != nil {
		writeDBError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
