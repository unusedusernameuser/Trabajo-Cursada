package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

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
