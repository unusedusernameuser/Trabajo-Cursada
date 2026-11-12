package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	db "my_sqlc_project/db/sqlc"
	"my_sqlc_project/handlers"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("la variable de entorno DATABASE_URL es obligatoria")
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("no se pudo abrir la conexión a la base de datos: %v", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		log.Fatalf("no se pudo conectar a la base de datos: %v", err)
	}

	handlers.Register(db.New(conn))

	port := ":8080"
	log.Printf("Servidor escuchando en http://localhost%s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Error al iniciar el servidor: %s\n", err)
	}
}
