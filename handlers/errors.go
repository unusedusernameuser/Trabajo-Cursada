package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

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
