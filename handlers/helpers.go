package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"

	"my_sqlc_project/views"
)

// parseID convierte un texto en un entero positivo de 32 bits.
func parseID(s string) (int32, bool) {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil || n <= 0 {
		return 0, false
	}
	return int32(n), true
}

// parseOptionalID es como parseID pero acepta el texto vacío (devuelve 0, "sin valor").
func parseOptionalID(s string) (int32, bool) {
	if s == "" {
		return 0, true
	}
	return parseID(s)
}

// pathID lee un parámetro entero positivo de la URL (por ejemplo {user_id}).
// Si no es válido ya responde 400.
func pathID(w http.ResponseWriter, r *http.Request, name string) (int32, bool) {
	id, ok := parseID(r.PathValue(name))
	if !ok {
		http.Error(w, name+" debe ser un entero positivo", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// pgCode devuelve el código SQLSTATE de un error de PostgreSQL ("" si no lo es).
func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

func serverError(w http.ResponseWriter, what string, err error) {
	log.Printf("%s: %v", what, err)
	http.Error(w, "error interno del servidor", http.StatusInternalServerError)
}

// checkText valida un texto obligatorio con largo máximo (mismas reglas que schema.sql).
// Devuelve "" si está bien o el mensaje de error.
func checkText(label, value string, max int) string {
	switch {
	case strings.TrimSpace(value) == "":
		return label + " es obligatorio"
	case utf8.RuneCountInString(value) > max:
		return label + " no puede superar los " + strconv.Itoa(max) + " caracteres"
	}
	return ""
}

func nullID(id int32) sql.NullInt32 {
	return sql.NullInt32{Int32: id, Valid: id != 0}
}

func nullText(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// parseDateTime interpreta el valor de <input type="datetime-local"> en la zona horaria
// de la aplicación. El texto vacío significa "sin fecha" (NULL).
func parseDateTime(s string) (sql.NullTime, bool) {
	if s == "" {
		return sql.NullTime{}, true
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, s, views.Location); err == nil {
			return sql.NullTime{Time: t, Valid: true}, true
		}
	}
	return sql.NullTime{}, false
}

// backFilters recupera los filtros que tenía la página al enviar el formulario (campo
// oculto "back"). Solo se usan los filtros conocidos: lo que mande el cliente nunca se
// copia tal cual a la URL del redirect.
func backFilters(r *http.Request) views.TaskFilters {
	v, _ := url.ParseQuery(r.PostFormValue("back"))
	return views.FiltersFromValues(v)
}

// redirectToTasks termina una acción (PRG): vuelve a la lista de tareas con los mismos filtros.
func redirectToTasks(w http.ResponseWriter, r *http.Request, userID int32) {
	http.Redirect(w, r, views.TasksURL(userID, backFilters(r)), http.StatusSeeOther)
}
