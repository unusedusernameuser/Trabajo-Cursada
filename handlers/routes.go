// Package handlers contiene los handlers HTTP de la aplicación. Ya no devuelven JSON:
// responden con HTML renderizado en el servidor con los componentes de views/.
package handlers

import (
	"net/http"

	db "my_sqlc_project/db/sqlc"
	"my_sqlc_project/views"
)

// Register registra en el mux por defecto de net/http las rutas de la aplicación.
func Register(queries *db.Queries) {
	h := &userHandler{queries: queries}

	// Lista + formulario
	http.HandleFunc("GET /{$}", h.index)
	// Alta y baja con HTMX: devuelven fragmentos de HTML, sin redirect
	http.HandleFunc("POST /users", h.create)
	http.HandleFunc("DELETE /users/{id}", h.delete)

	registerTaskRoutes(queries)

	// Cualquier otra ruta
	http.HandleFunc("/", notFound)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_ = views.NotFoundPage().Render(r.Context(), w)
}

// registerTaskRoutes registra la página de tareas de cada usuario. Todas las rutas cuelgan de
// /users/{user_id}. Los formularios HTML solo soportan GET y POST: toda acción que modifica
// algo es un POST que termina en un redirect (303) a la lista de tareas.
func registerTaskRoutes(queries *db.Queries) {
	t := &tasksHandler{queries: queries}

	// Página: lista, filtros (query string) y formulario de alta
	http.HandleFunc("GET /users/{user_id}/tasks", t.index)

	// Tareas
	http.HandleFunc("POST /users/{user_id}/tasks", t.create)
	http.HandleFunc("GET /users/{user_id}/tasks/{id}/edit", t.editForm)
	http.HandleFunc("POST /users/{user_id}/tasks/{id}", t.update)
	http.HandleFunc("POST /users/{user_id}/tasks/{id}/delete", t.delete)
	http.HandleFunc("POST /users/{user_id}/tasks/{id}/status", t.updateStatus)
	http.HandleFunc("POST /users/{user_id}/tasks/{id}/folder", t.moveToFolder)

	// Tags de una tarea
	http.HandleFunc("POST /users/{user_id}/tasks/{id}/tags", t.addTag)
	http.HandleFunc("POST /users/{user_id}/tasks/{id}/tags/{tag_id}/delete", t.removeTag)

	// Carpetas
	http.HandleFunc("POST /users/{user_id}/folders", t.createFolder)
	http.HandleFunc("POST /users/{user_id}/folders/{id}", t.renameFolder)
	http.HandleFunc("POST /users/{user_id}/folders/{id}/delete", t.deleteFolder)

	// Tags
	http.HandleFunc("POST /users/{user_id}/tags", t.createTag)
	http.HandleFunc("POST /users/{user_id}/tags/{id}", t.renameTag)
	http.HandleFunc("POST /users/{user_id}/tags/{id}/delete", t.deleteTag)
}
