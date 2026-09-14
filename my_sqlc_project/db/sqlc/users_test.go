package db

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func setupTx(t *testing.T) (*Queries, func()) {
	t.Helper()

	connStr := "host=localhost port=5432 user=rossi password=277353 dbname=webapp_db"

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("no se pudo conectar a la DB: %v", err)
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		db.Close()
		t.Fatalf("no se pudo iniciar la transacción: %v", err)
	}

	queries := New(tx)

	cleanup := func() {
		_ = tx.Rollback()
		_ = db.Close()
	}

	return queries, cleanup
}

func createTestUser(t *testing.T, q *Queries) User {
	t.Helper()

	ctx := context.Background()
	user, err := q.CreateUser(ctx, CreateUserParams{
		Handle:       "testuser",
		DisplayName:  "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password_123",
	})
	if err != nil {
		t.Fatalf("CreateUser falló: %v", err)
	}
	return user
}

func TestCreateUser(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	user := createTestUser(t, q)

	if user.ID == 0 {
		t.Error("se esperaba un ID distinto de 0")
	}
	if user.Handle != "testuser" {
		t.Errorf("handle: se esperaba %q, se obtuvo %q", "testuser", user.Handle)
	}
	if user.Email != "test@example.com" {
		t.Errorf("email: se esperaba %q, se obtuvo %q", "test@example.com", user.Email)
	}
}

func TestCreateUser_NormalizaHandleYEmail(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	ctx := context.Background()
	user, err := q.CreateUser(ctx, CreateUserParams{
		Handle:       "MayusHandle",
		DisplayName:  "Con Mayus",
		Email:        "Mayus@Example.com",
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("CreateUser falló: %v", err)
	}

	if user.Handle != "mayushandle" {
		t.Errorf("se esperaba que el handle se guarde en minúsculas, se obtuvo %q", user.Handle)
	}
	if user.Email != "mayus@example.com" {
		t.Errorf("se esperaba que el email se guarde en minúsculas, se obtuvo %q", user.Email)
	}
}

func TestGetUserByID(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	created := createTestUser(t, q)

	got, err := q.GetUserByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetUserByID falló: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID: se esperaba %d, se obtuvo %d", created.ID, got.ID)
	}
}

func TestGetUserByHandle(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	created := createTestUser(t, q)

	got, err := q.GetUserByHandle(context.Background(), created.Handle)
	if err != nil {
		t.Fatalf("GetUserByHandle falló: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID: se esperaba %d, se obtuvo %d", created.ID, got.ID)
	}
}

func TestGetUserByEmail(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	created := createTestUser(t, q)

	got, err := q.GetUserByEmail(context.Background(), created.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail falló: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID: se esperaba %d, se obtuvo %d", created.ID, got.ID)
	}
}

func TestGetUserAuthByEmail(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	created := createTestUser(t, q)

	auth, err := q.GetUserAuthByEmail(context.Background(), created.Email)
	if err != nil {
		t.Fatalf("GetUserAuthByEmail falló: %v", err)
	}
	if auth.ID != created.ID {
		t.Errorf("ID: se esperaba %d, se obtuvo %d", created.ID, auth.ID)
	}
	if auth.PasswordHash != "hashed_password_123" {
		t.Errorf("password_hash inesperado: %q", auth.PasswordHash)
	}
}

func TestListUsers(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	createTestUser(t, q)

	ctx := context.Background()
	_, err := q.CreateUser(ctx, CreateUserParams{
		Handle:       "otrousuario",
		DisplayName:  "Otro Usuario",
		Email:        "otro@example.com",
		PasswordHash: "hash2",
	})
	if err != nil {
		t.Fatalf("CreateUser (segundo usuario) falló: %v", err)
	}

	users, err := q.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers falló: %v", err)
	}
	if len(users) < 2 {
		t.Errorf("se esperaban al menos 2 usuarios, se obtuvieron %d", len(users))
	}
}

func TestUpdateDisplayName(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	created := createTestUser(t, q)

	updated, err := q.UpdateDisplayName(context.Background(), UpdateDisplayNameParams{
		ID:          created.ID,
		DisplayName: "Nuevo Nombre",
	})
	if err != nil {
		t.Fatalf("UpdateDisplayName falló: %v", err)
	}
	if updated.DisplayName != "Nuevo Nombre" {
		t.Errorf("display_name: se esperaba %q, se obtuvo %q", "Nuevo Nombre", updated.DisplayName)
	}
}

func TestUpdateHandle(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	created := createTestUser(t, q)

	updated, err := q.UpdateHandle(context.Background(), UpdateHandleParams{
		ID:     created.ID,
		Handle: "NuevoHandle",
	})
	if err != nil {
		t.Fatalf("UpdateHandle falló: %v", err)
	}
	if updated.Handle != "nuevohandle" {
		t.Errorf("handle: se esperaba %q (en minúsculas), se obtuvo %q", "nuevohandle", updated.Handle)
	}
}

func TestUpdateEmail(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	created := createTestUser(t, q)

	updated, err := q.UpdateEmail(context.Background(), UpdateEmailParams{
		ID:    created.ID,
		Email: "Nuevo@Example.com",
	})
	if err != nil {
		t.Fatalf("UpdateEmail falló: %v", err)
	}
	if updated.Email != "nuevo@example.com" {
		t.Errorf("email: se esperaba %q (en minúsculas), se obtuvo %q", "nuevo@example.com", updated.Email)
	}
}

func TestUpdatePassword(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	created := createTestUser(t, q)
	ctx := context.Background()

	err := q.UpdatePassword(ctx, UpdatePasswordParams{
		ID:           created.ID,
		PasswordHash: "nuevo_hash",
	})
	if err != nil {
		t.Fatalf("UpdatePassword falló: %v", err)
	}

	auth, err := q.GetUserAuthByEmail(ctx, created.Email)
	if err != nil {
		t.Fatalf("GetUserAuthByEmail falló: %v", err)
	}
	if auth.PasswordHash != "nuevo_hash" {
		t.Errorf("password_hash: se esperaba %q, se obtuvo %q", "nuevo_hash", auth.PasswordHash)
	}
}

func TestDeleteUser(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	created := createTestUser(t, q)
	ctx := context.Background()

	if err := q.DeleteUser(ctx, created.ID); err != nil {
		t.Fatalf("DeleteUser falló: %v", err)
	}

	_, err := q.GetUserByID(ctx, created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("se esperaba sql.ErrNoRows tras borrar, se obtuvo: %v", err)
	}
}

func TestCreateUser_HandleDuplicadoFalla(t *testing.T) {
	q, cleanup := setupTx(t)
	defer cleanup()

	createTestUser(t, q)

	ctx := context.Background()
	_, err := q.CreateUser(ctx, CreateUserParams{
		Handle:       "testuser",
		DisplayName:  "Otro Nombre",
		Email:        "otro2@example.com",
		PasswordHash: "hash3",
	})
	if err == nil {
		t.Error("se esperaba un error por handle duplicado (constraint UNIQUE), pero no hubo error")
	}
}
