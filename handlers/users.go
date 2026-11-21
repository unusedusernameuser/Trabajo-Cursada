package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

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
// Con HTMX ya no hay redirección: después de guardar se vuelve a consultar la lista completa
// y se devuelve únicamente el componente UserList, que HTMX coloca en el target (#user-list).
// Se agrega un formulario vacío marcado como OOB (hx-swap-oob) para limpiar los campos.
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
		h.renderFormError(w, r, http.StatusBadRequest, form)
		return
	}

	_, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Handle:       form.Handle,
		DisplayName:  form.DisplayName,
		Email:        form.Email,
		PasswordHash: password,
	})
	if err != nil {
		if pgCode(err) == pgUniqueViolation {
			form.Error = "ya existe un usuario con ese handle o email"
			h.renderFormError(w, r, http.StatusConflict, form)
			return
		}
		log.Printf("error al crear usuario: %v", err)
		http.Error(w, "error interno del servidor", http.StatusInternalServerError)
		return
	}

	// Se vuelve a consultar la lista completa y se devuelve solo ese fragmento.
	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		log.Printf("error al listar usuarios: %v", err)
		http.Error(w, "error interno del servidor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.UserList(users).Render(r.Context(), w); err != nil {
		log.Printf("error al renderizar la lista: %v", err)
		return
	}
	// Formulario limpio, intercambiado fuera de banda.
	if err := views.UserForm(views.UserFormValues{OOB: true}).Render(r.Context(), w); err != nil {
		log.Printf("error al renderizar el formulario: %v", err)
	}
}

// renderFormError responde a un alta rechazada (400/409) con el formulario y su mensaje.
// El target del formulario es #user-list, así que HX-Retarget / HX-Reswap indican a HTMX
// que reemplace #user-form (el formulario) y no la lista.
func (h *userHandler) renderFormError(w http.ResponseWriter, r *http.Request, status int, form views.UserFormValues) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Retarget", "#user-form")
	w.Header().Set("HX-Reswap", "outerHTML")
	w.WriteHeader(status)
	if err := views.UserForm(form).Render(r.Context(), w); err != nil {
		log.Printf("error al renderizar el formulario: %v", err)
	}
}

// DELETE /users/{id}
// Borra el usuario y responde 200 sin contenido para el elemento: HTMX reemplaza el target
// (la fila, "closest tr") por nada y la fila desaparece del DOM. Lo único que viaja es el
// contador del título, marcado hx-swap-oob, para que no quede desactualizado.
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

	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		// El usuario ya se borró: la fila debe desaparecer igual (200 vacío).
		log.Printf("error al listar usuarios: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.UserCountOOB(len(users)).Render(r.Context(), w); err != nil {
		log.Printf("error al renderizar el contador: %v", err)
	}
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
