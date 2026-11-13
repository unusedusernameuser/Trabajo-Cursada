package db

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
)

// Helpers compartidos por los tests de db/sqlc. Todo lo que cuelga de un usuario
// (carpetas, tags, tareas) se elimina en cascada al borrar al usuario,
// así que alcanza con registrar el DeleteUser en t.Cleanup.

func uniqueSuffix(prefix string) string {
	return fmt.Sprintf("%s_%06d", prefix, rand.Intn(1000000))
}

// createTestUser crea un usuario y lo elimina (junto con sus datos) al terminar el test.
// El prefijo debe ser corto: el handle tiene un máximo de 31 caracteres.
func createTestUser(t *testing.T, q *Queries, prefix string) CreateUserRow {
	t.Helper()
	ctx := context.Background()

	u, err := q.CreateUser(ctx, randomUserParams(uniqueSuffix(prefix)))
	if err != nil {
		t.Fatalf("CreateUser falló: %v", err)
	}
	t.Cleanup(func() {
		_ = q.DeleteUser(ctx, u.ID)
	})
	return u
}

func createTestFolder(t *testing.T, q *Queries, userID int32, name string) Folder {
	t.Helper()

	f, err := q.CreateFolder(context.Background(), CreateFolderParams{
		UserID:     userID,
		FolderName: name,
	})
	if err != nil {
		t.Fatalf("CreateFolder falló: %v", err)
	}
	return f
}

func createTestTag(t *testing.T, q *Queries, userID int32, name string) Tag {
	t.Helper()

	tag, err := q.CreateTag(context.Background(), CreateTagParams{
		UserID:  userID,
		TagName: name,
	})
	if err != nil {
		t.Fatalf("CreateTag falló: %v", err)
	}
	return tag
}

// createTestTask crea una tarea simple (sin carpeta ni fecha).
func createTestTask(t *testing.T, q *Queries, userID int32, title string) Task {
	t.Helper()

	task, err := q.CreateTask(context.Background(), CreateTaskParams{
		UserID: userID,
		Title:  title,
	})
	if err != nil {
		t.Fatalf("CreateTask falló: %v", err)
	}
	return task
}
