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
	// Alta y baja: ambas terminan en un redirect a /
	http.HandleFunc("POST /users", h.create)
	http.HandleFunc("POST /users/{id}/delete", h.delete)
	// Cualquier otra ruta
	http.HandleFunc("/", notFound)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_ = views.NotFoundPage().Render(r.Context(), w)
}
