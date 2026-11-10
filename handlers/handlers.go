package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
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
		writeUserDBError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

// GET /users
func (h *userHandler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		writeUserDBError(w, err)
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
		writeUserDBError(w, err)
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
		writeUserDBError(w, err)
		return
	}
	defer tx.Rollback()
	q := h.queries.WithTx(tx)

	if _, err := q.UpdateHandle(ctx, db.UpdateHandleParams{ID: id, Handle: in.Handle}); err != nil {
		writeUserDBError(w, err)
		return
	}
	if _, err := q.UpdateEmail(ctx, db.UpdateEmailParams{ID: id, Email: in.Email}); err != nil {
		writeUserDBError(w, err)
		return
	}
	user, err := q.UpdateDisplayName(ctx, db.UpdateDisplayNameParams{ID: id, DisplayName: in.DisplayName})
	if err != nil {
		writeUserDBError(w, err)
		return
	}

	if err := tx.Commit(); err != nil {
		writeUserDBError(w, err)
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
		writeUserDBError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// readID obtiene el {id} de la URL. Si no es un entero positivo responde 400.
func readID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	return readPathID(w, r, "id")
}

// readPathID obtiene un parámetro entero positivo de la URL (por ejemplo {user_id}).
// Si no es válido responde 400.
func readPathID(w http.ResponseWriter, r *http.Request, name string) (int32, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 32)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, name+" debe ser un entero positivo")
		return 0, false
	}
	return int32(id), true
}

// parsePositiveInt32 convierte un texto (por ejemplo de un query param) a un entero positivo.
func parsePositiveInt32(s string) (int32, bool) {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil || n <= 0 {
		return 0, false
	}
	return int32(n), true
}

// isNoRows indica si la query no devolvió filas.
func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// decodeBody decodifica el cuerpo JSON en dst. Si falla ya responde 400.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "el cuerpo debe ser un JSON válido")
		return false
	}
	return true
}

// checkText valida un texto obligatorio con largo máximo (mismas reglas que schema.sql).
// Si no cumple ya responde 400.
func checkText(w http.ResponseWriter, field, value string, max int) bool {
	switch {
	case strings.TrimSpace(value) == "":
		writeError(w, http.StatusBadRequest, field+" es obligatorio")
	case utf8.RuneCountInString(value) > max:
		writeError(w, http.StatusBadRequest, field+" no puede superar los "+strconv.Itoa(max)+" caracteres")
	default:
		return true
	}
	return false
}

// Conversiones entre los tipos sql.Null* de sqlc y punteros, para que el JSON
// muestre el valor o null (y no {"String": ..., "Valid": ...}).
func nullTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func nullString(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func nullInt32(n sql.NullInt32) *int32 {
	if !n.Valid {
		return nil
	}
	return &n.Int32
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func toNullInt32(n *int32) sql.NullInt32 {
	if n == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *n, Valid: true}
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

// writeDBError traduce errores de la base de datos a una respuesta HTTP genérica.
func writeDBError(w http.ResponseWriter, err error) {
	writeDBErrorMsg(w, err, "recurso no encontrado", "ya existe un registro con esos datos")
}

// writeUserDBError es writeDBError con los mensajes de la entidad users.
func writeUserDBError(w http.ResponseWriter, err error) {
	writeDBErrorMsg(w, err, "usuario no encontrado", "ya existe un usuario con ese handle o email")
}

// writeDBErrorMsg traduce errores de la base de datos con mensajes a medida
// para el "no encontrado" (404) y el "duplicado" (409).
func writeDBErrorMsg(w http.ResponseWriter, err error, notFound, conflict string) {
	var pgErr *pgconn.PgError

	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, notFound)
	case errors.As(err, &pgErr) && pgErr.Code == "23505": // unique_violation
		writeError(w, http.StatusConflict, conflict)
	case errors.As(err, &pgErr) && pgErr.Code == "23503": // foreign_key_violation
		writeError(w, http.StatusNotFound, "el recurso referenciado no existe")
	case errors.As(err, &pgErr) && pgErr.Code == "23514": // check_violation
		writeError(w, http.StatusBadRequest, "los datos no cumplen las restricciones de la base de datos")
	default:
		log.Printf("error de base de datos: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno del servidor")
	}
}
