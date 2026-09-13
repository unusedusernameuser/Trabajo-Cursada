package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	sqlc "my_sqlc_project/db/sqlc"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	runUserDemo()
	startStaticServer()
}

func runUserDemo() {
	connStr := "host=localhost port=5432 user=rossi password=277353 dbname=webapp_db"

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}
	defer db.Close()

	queries := sqlc.New(db)
	ctx := context.Background()

	createdUser, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
		Handle:       "jdoe",
		DisplayName:  "John Doe",
		Email:        "john.doe@example.com",
		PasswordHash: "passwordexample",
	})
	if err != nil {
		log.Fatalf("failed to create user: %v", err)
	}
	log.Printf("Created user: %+v\n", createdUser)

	user, err := queries.GetUserByID(ctx, createdUser.ID)
	if err != nil {
		log.Fatalf("failed to get user by id: %v", err)
	}
	log.Printf("Retrieved user by id: %+v\n", user)

	userByHandle, err := queries.GetUserByHandle(ctx, createdUser.Handle)
	if err != nil {
		log.Fatalf("failed to get user by handle: %v", err)
	}
	log.Printf("Retrieved user by handle: %+v\n", userByHandle)

	userByEmail, err := queries.GetUserByEmail(ctx, createdUser.Email)
	if err != nil {
		log.Fatalf("failed to get user by email: %v", err)
	}
	log.Printf("Retrieved user by email: %+v\n", userByEmail)

	authUser, err := queries.GetUserAuthByEmail(ctx, createdUser.Email)
	if err != nil {
		log.Fatalf("failed to get user auth by email: %v", err)
	}
	log.Printf("Retrieved auth record for longin check: %v\n", authUser)

	users, err := queries.ListUsers(ctx)
	if err != nil {
		log.Fatalf("failed to list users: %v", err)
	}
	log.Printf("All users: %+v\n", users)

	updatedDisplayName, err := queries.UpdateDisplayName(ctx, sqlc.UpdateDisplayNameParams{
		ID:          createdUser.ID,
		DisplayName: "Johnny Doe",
	})
	if err != nil {
		log.Fatalf("failed to update display name: %v", err)
	}
	log.Printf("Updated display name: %+v\n", updatedDisplayName)

	updatedHandle, err := queries.UpdateHandle(ctx, sqlc.UpdateHandleParams{
		ID:     createdUser.ID,
		Handle: "johnnyd",
	})
	if err != nil {
		log.Fatalf("failed to update handle: %v", err)
	}
	log.Printf("Updated handle: %+v\n", updatedHandle)

	updatedEmail, err := queries.UpdateEmail(ctx, sqlc.UpdateEmailParams{
		ID:    createdUser.ID,
		Email: "johnny.doe@example.com",
	})
	if err != nil {
		log.Fatalf("failed to update email: %v", err)
	}
	log.Printf("Updated email: %+v\n", updatedEmail)

	err = queries.UpdatePassword(ctx, sqlc.UpdatePasswordParams{
		ID:           createdUser.ID,
		PasswordHash: "anotherpassword",
	})
	if err != nil {
		log.Fatalf("failed to update password: %v", err)
	}
	log.Println("Password updated successfully")

	finalUser, err := queries.GetUserByID(ctx, createdUser.ID)
	if err != nil {
		log.Fatalf("failed to get updated user: %v", err)
	}
	log.Printf("Final user state: %+v\n", finalUser)

	err = queries.DeleteUser(ctx, createdUser.ID)
	if err != nil {
		log.Fatalf("failed to delete user: %v", err)
	}
	log.Println("User deleted successfully")

	_, err = queries.GetUserByID(ctx, createdUser.ID)
	if err == sql.ErrNoRows {
		log.Println("User not found after deletion")
	} else if err != nil {
		log.Fatalf("failed to get user after deletion: %v", err)
	}
}

func startStaticServer() {
	staticDir := "./static"
	fileServer := http.FileServer(http.Dir(staticDir))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(staticDir, filepath.Clean(r.URL.Path))

		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			serve404(w, staticDir)
			return
		}
		if err != nil {
			log.Printf("error al acceder a %s: %v", path, err)
			http.Error(w, "error interno del servidor", http.StatusInternalServerError)
			return
		}

		if info.IsDir() && r.URL.Path != "/" {
			indexPath := filepath.Join(path, "index.html")
			if _, errIndex := os.Stat(indexPath); os.IsNotExist(errIndex) {
				serve404(w, staticDir)
				return
			}
		}

		fileServer.ServeHTTP(w, r)
	})

	port := ":8080"
	log.Printf("Servidor estatico escuchando en http://localhost%s\n", port)
	log.Printf("Sirviendo archivos desde: %s\n", staticDir)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Error al iniciar el servidor: %s\n", err)
	}
}

func serve404(w http.ResponseWriter, staticDir string) {
	content, err := os.ReadFile(filepath.Join(staticDir, "404.html"))
	if err != nil {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write(content)
}
