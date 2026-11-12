// Package views contiene los componentes templ que generan el HTML de la aplicación.
package views

import (
	"database/sql"
	"strconv"
)

// UserFormValues son los datos con los que se dibuja el formulario de alta.
// Si el servidor rechaza el envío, se vuelve a mostrar con lo que escribió la
// persona (salvo la contraseña) y el mensaje de error.
type UserFormValues struct {
	Handle      string
	DisplayName string
	Email       string
	Error       string
}

// formatDate muestra una fecha dd/mm/aaaa o vacío si la columna es NULL.
func formatDate(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("02/01/2006")
}

func userDeleteURL(id int32) string {
	return "/users/" + strconv.Itoa(int(id)) + "/delete"
}
