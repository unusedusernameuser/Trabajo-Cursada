package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// getTestDB abre una conexión a la base de datos usada en los tests.
// Usa la variable de entorno DATABASE_URL (definida en .env / docker-compose).
func getTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/my_sqlc_project?sslmode=disable"
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("no se pudo abrir la conexión: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		t.Fatalf("no se pudo hacer ping a la base de datos: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
	})

	return conn
}

func randomUserParams(suffix string) CreateUserParams {
	return CreateUserParams{
		Handle:       fmt.Sprintf("test_user_%s", suffix),
		DisplayName:  fmt.Sprintf("Test User %s", suffix),
		Email:        fmt.Sprintf("test_user_%s@example.com", suffix),
		PasswordHash: "not_a_real_hash",
	}
}

func TestCreateAndGetUser(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	params := randomUserParams("create_get")

	created, err := q.CreateUser(ctx, params)
	if err != nil {
		t.Fatalf("CreateUser falló: %v", err)
	}
	t.Cleanup(func() {
		_ = q.DeleteUser(ctx, created.ID)
	})

	if created.Handle != params.Handle {
		t.Errorf("handle = %q, quería %q", created.Handle, params.Handle)
	}
	if created.Email != params.Email {
		t.Errorf("email = %q, quería %q", created.Email, params.Email)
	}

	byID, err := q.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID falló: %v", err)
	}
	if byID.Handle != params.Handle {
		t.Errorf("GetUserByID: handle = %q, quería %q", byID.Handle, params.Handle)
	}

	byHandle, err := q.GetUserByHandle(ctx, params.Handle)
	if err != nil {
		t.Fatalf("GetUserByHandle falló: %v", err)
	}
	if byHandle.ID != created.ID {
		t.Errorf("GetUserByHandle: id = %d, quería %d", byHandle.ID, created.ID)
	}

	byEmail, err := q.GetUserByEmail(ctx, params.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail falló: %v", err)
	}
	if byEmail.ID != created.ID {
		t.Errorf("GetUserByEmail: id = %d, quería %d", byEmail.ID, created.ID)
	}
}

func TestUpdateUser(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	created, err := q.CreateUser(ctx, randomUserParams("update"))
	if err != nil {
		t.Fatalf("CreateUser falló: %v", err)
	}
	t.Cleanup(func() {
		_ = q.DeleteUser(ctx, created.ID)
	})

	updated, err := q.UpdateDisplayName(ctx, UpdateDisplayNameParams{
		ID:          created.ID,
		DisplayName: "Nuevo Nombre",
	})
	if err != nil {
		t.Fatalf("UpdateDisplayName falló: %v", err)
	}
	if updated.DisplayName != "Nuevo Nombre" {
		t.Errorf("display_name = %q, quería %q", updated.DisplayName, "Nuevo Nombre")
	}

	updatedHandle, err := q.UpdateHandle(ctx, UpdateHandleParams{
		ID:     created.ID,
		Handle: "nuevo_handle",
	})
	if err != nil {
		t.Fatalf("UpdateHandle falló: %v", err)
	}
	if updatedHandle.Handle != "nuevo_handle" {
		t.Errorf("handle = %q, quería %q", updatedHandle.Handle, "nuevo_handle")
	}
}

func TestDeleteUser(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	created, err := q.CreateUser(ctx, randomUserParams("delete"))
	if err != nil {
		t.Fatalf("CreateUser falló: %v", err)
	}

	if err := q.DeleteUser(ctx, created.ID); err != nil {
		t.Fatalf("DeleteUser falló: %v", err)
	}

	if _, err := q.GetUserByID(ctx, created.ID); err == nil {
		t.Errorf("se esperaba un error al buscar un usuario eliminado, pero no hubo ninguno")
	}
}

func TestListUsers(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	created, err := q.CreateUser(ctx, randomUserParams("list"))
	if err != nil {
		t.Fatalf("CreateUser falló: %v", err)
	}
	t.Cleanup(func() {
		_ = q.DeleteUser(ctx, created.ID)
	})

	users, err := q.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers falló: %v", err)
	}

	found := false
	for _, u := range users {
		if u.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("el usuario creado no aparece en ListUsers")
	}
}
