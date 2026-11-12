package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"

	db "my_sqlc_project/db/sqlc"
	"my_sqlc_project/views"
)

// userHandler agrupa los handlers de la entidad users.
type userHandler struct {
	queries *db.Queries
}

// GET /
// Obtiene todos los usuarios con ListUsers y renderiza la página completa.
func (h *userHandler) index(w http.ResponseWriter, r *http.Request) {
	h.renderIndex(w, r, http.StatusOK, views.UserFormValues{})
}

// renderIndex lee la lista de la base y escribe IndexPage en el ResponseWriter.
// Se usa también para volver a mostrar el formulario con un error (400/409).
func (h *userHandler) renderIndex(w http.ResponseWriter, r *http.Request, status int, form views.UserFormValues) {
	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		log.Printf("error al listar usuarios: %v", err)
		http.Error(w, "error interno del servidor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := views.IndexPage(users, form).Render(r.Context(), w); err != nil {
		log.Printf("error al renderizar la página: %v", err)
	}
}

// POST /users
// Patrón Post/Redirect/Get: si se guarda, redirige a GET / para que el navegador
// pida la página de nuevo (así recargarla no reenvía el formulario).
func (h *userHandler) create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}

	form := views.UserFormValues{
		Handle:      strings.TrimSpace(r.PostFormValue("handle")),
		DisplayName: strings.TrimSpace(r.PostFormValue("display_name")),
		Email:       strings.TrimSpace(r.PostFormValue("email")),
	}
	// Igual que en la entrega anterior, la contraseña se guarda tal cual en password_hash;
	// en una aplicación real se hashearía acá (por ejemplo con bcrypt) antes de guardarla.
	password := r.PostFormValue("password")

	if msg := validateUser(form, password); msg != "" {
		form.Error = msg
		h.renderIndex(w, r, http.StatusBadRequest, form)
		return
	}

	_, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Handle:       form.Handle,
		DisplayName:  form.DisplayName,
		Email:        form.Email,
		PasswordHash: password,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			form.Error = "ya existe un usuario con ese handle o email"
			h.renderIndex(w, r, http.StatusConflict, form)
			return
		}
		log.Printf("error al crear usuario: %v", err)
		http.Error(w, "error interno del servidor", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// POST /users/{id}/delete
// Los formularios HTML solo soportan GET y POST, por eso el borrado es un POST.
// También termina con un redirect a /.
func (h *userHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id <= 0 {
		http.Error(w, "id debe ser un entero positivo", http.StatusBadRequest)
		return
	}

	if err := h.queries.DeleteUser(r.Context(), int32(id)); err != nil {
		log.Printf("error al eliminar usuario: %v", err)
		http.Error(w, "error interno del servidor", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// validateUser aplica las mismas reglas que schema.sql: no vacío y largo máximo.
// Devuelve "" si todo está bien o el mensaje de error.
func validateUser(f views.UserFormValues, password string) string {
	switch {
	case f.Handle == "":
		return "el nombre de usuario es obligatorio"
	case utf8.RuneCountInString(f.Handle) > 31:
		return "el nombre de usuario no puede superar los 31 caracteres"
	case f.DisplayName == "":
		return "el nombre para mostrar es obligatorio"
	case utf8.RuneCountInString(f.DisplayName) > 63:
		return "el nombre para mostrar no puede superar los 63 caracteres"
	case f.Email == "":
		return "el email es obligatorio"
	case utf8.RuneCountInString(f.Email) > 255:
		return "el email no puede superar los 255 caracteres"
	case strings.TrimSpace(password) == "":
		return "la contraseña es obligatoria"
	case utf8.RuneCountInString(password) > 255:
		return "la contraseña no puede superar los 255 caracteres"
	}
	return ""
}
