package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"

	db "my_sqlc_project/db/sqlc"
)

// userHandler agrupa los handlers CRUD de la entidad users.
// conn se usa solo para abrir la transacción del PUT.
type userHandler struct {
	conn    *sql.DB
	queries *db.Queries
}

// userInput es el cuerpo JSON que reciben POST y PUT.
type userInput struct {
	Handle       string `json:"handle"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

func registerUserRoutes(conn *sql.DB, queries *db.Queries) {
	h := &userHandler{conn: conn, queries: queries}

	http.HandleFunc("POST /users", h.create)
	http.HandleFunc("GET /users", h.list)
	http.HandleFunc("GET /users/{id}", h.get)
	http.HandleFunc("PUT /users/{id}", h.update)
	http.HandleFunc("DELETE /users/{id}", h.delete)
}

// POST /users
func (h *userHandler) create(w http.ResponseWriter, r *http.Request) {
	in, ok := readUserInput(w, r, true)
	if !ok {
		return
	}

	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Handle:       in.Handle,
		DisplayName:  in.DisplayName,
		Email:        in.Email,
		PasswordHash: in.PasswordHash,
	})
	if err != nil {
		writeDBError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

// GET /users
func (h *userHandler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		writeDBError(w, err)
		return
	}
	if users == nil {
		users = []db.ListUsersRow{} // para responder [] y no null
	}

	writeJSON(w, http.StatusOK, users)
}

// GET /users/{id}
func (h *userHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), id)
	if err != nil {
		writeDBError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// PUT /users/{id}
func (h *userHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	in, ok := readUserInput(w, r, false)
	if !ok {
		return
	}

	ctx := r.Context()

	tx, err := h.conn.BeginTx(ctx, nil)
	if err != nil {
		writeDBError(w, err)
		return
	}
	defer tx.Rollback()
	q := h.queries.WithTx(tx)

	if _, err := q.UpdateHandle(ctx, db.UpdateHandleParams{ID: id, Handle: in.Handle}); err != nil {
		writeDBError(w, err)
		return
	}
	if _, err := q.UpdateEmail(ctx, db.UpdateEmailParams{ID: id, Email: in.Email}); err != nil {
		writeDBError(w, err)
		return
	}
	user, err := q.UpdateDisplayName(ctx, db.UpdateDisplayNameParams{ID: id, DisplayName: in.DisplayName})
	if err != nil {
		writeDBError(w, err)
		return
	}

	if err := tx.Commit(); err != nil {
		writeDBError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// DELETE /users/{id}
func (h *userHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}

	if err := h.queries.DeleteUser(r.Context(), id); err != nil {
		writeDBError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// readID obtiene el {id} de la URL. Si no es un entero positivo responde 400.
func readID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "el id debe ser un entero positivo")
		return 0, false
	}
	return int32(id), true
}

// readUserInput decodifica y valida el cuerpo JSON. Si algo falla ya responde 400.
// requirePassword indica si password_hash es obligatorio (POST) o se ignora (PUT).
func readUserInput(w http.ResponseWriter, r *http.Request, requirePassword bool) (userInput, bool) {
	var in userInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "el cuerpo debe ser un JSON válido")
		return in, false
	}

	// Mismas reglas que schema.sql: no vacío y largo máximo de cada columna.
	switch {
	case strings.TrimSpace(in.Handle) == "":
		writeError(w, http.StatusBadRequest, "handle es obligatorio")
	case utf8.RuneCountInString(in.Handle) > 31:
		writeError(w, http.StatusBadRequest, "handle no puede superar los 31 caracteres")
	case strings.TrimSpace(in.DisplayName) == "":
		writeError(w, http.StatusBadRequest, "display_name es obligatorio")
	case utf8.RuneCountInString(in.DisplayName) > 63:
		writeError(w, http.StatusBadRequest, "display_name no puede superar los 63 caracteres")
	case strings.TrimSpace(in.Email) == "":
		writeError(w, http.StatusBadRequest, "email es obligatorio")
	case utf8.RuneCountInString(in.Email) > 255:
		writeError(w, http.StatusBadRequest, "email no puede superar los 255 caracteres")
	case requirePassword && strings.TrimSpace(in.PasswordHash) == "":
		writeError(w, http.StatusBadRequest, "password_hash es obligatorio")
	case requirePassword && utf8.RuneCountInString(in.PasswordHash) > 255:
		writeError(w, http.StatusBadRequest, "password_hash no puede superar los 255 caracteres")
	default:
		return in, true
	}
	return in, false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error al escribir la respuesta: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeDBError traduce errores de la base de datos a una respuesta HTTP.
func writeDBError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError

	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "usuario no encontrado")
	case errors.As(err, &pgErr) && pgErr.Code == "23505": // unique_violation
		writeError(w, http.StatusConflict, "ya existe un usuario con ese handle o email")
	default:
		log.Printf("error de base de datos: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno del servidor")
	}
}
