// Package handlers expone la API REST: un archivo por entidad (users, folders, tags,
// tasks) más utilidades compartidas (helpers.go y errors.go).
package handlers

import (
	"database/sql"

	db "my_sqlc_project/db/sqlc"
)

// Register registra en el mux por defecto de net/http las rutas de todas las entidades.
// conn se usa solo para las operaciones que necesitan abrir una transacción.
func Register(conn *sql.DB, queries *db.Queries) {
	registerUserRoutes(conn, queries)
	registerFolderRoutes(queries)
	registerTagRoutes(queries)
	registerTaskRoutes(queries)
}
