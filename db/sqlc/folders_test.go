package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestCreateAndGetFolder(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "fld_cg")

	created, err := q.CreateFolder(ctx, CreateFolderParams{UserID: user.ID, FolderName: "Trabajo"})
	if err != nil {
		t.Fatalf("CreateFolder falló: %v", err)
	}
	if created.FolderName != "Trabajo" {
		t.Errorf("folder_name = %q, quería %q", created.FolderName, "Trabajo")
	}
	if created.UserID != user.ID {
		t.Errorf("user_id = %d, quería %d", created.UserID, user.ID)
	}

	got, err := q.GetFolderByID(ctx, GetFolderByIDParams{ID: created.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("GetFolderByID falló: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetFolderByID: id = %d, quería %d", got.ID, created.ID)
	}

	// Otro usuario no puede ver la carpeta.
	other := createTestUser(t, q, "fld_cg2")
	_, err = q.GetFolderByID(ctx, GetFolderByIDParams{ID: created.ID, UserID: other.ID})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("se esperaba sql.ErrNoRows para otro usuario, se obtuvo: %v", err)
	}
}

func TestCreateFolderDuplicate(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "fld_dup")
	createTestFolder(t, q, user.ID, "Repetida")

	_, err := q.CreateFolder(ctx, CreateFolderParams{UserID: user.ID, FolderName: "Repetida"})
	if err == nil {
		t.Errorf("se esperaba un error por (user_id, folder_name) duplicado, pero no hubo ninguno")
	}
}

func TestListFoldersByUser(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "fld_list")
	other := createTestUser(t, q, "fld_list2")

	createTestFolder(t, q, user.ID, "B carpeta")
	createTestFolder(t, q, user.ID, "A carpeta")
	createTestFolder(t, q, other.ID, "Ajena")

	folders, err := q.ListFoldersByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListFoldersByUser falló: %v", err)
	}
	if len(folders) != 2 {
		t.Fatalf("len(folders) = %d, quería 2", len(folders))
	}
	// ORDER BY folder_name
	if folders[0].FolderName != "A carpeta" || folders[1].FolderName != "B carpeta" {
		t.Errorf("orden incorrecto: %q, %q", folders[0].FolderName, folders[1].FolderName)
	}
}

func TestUpdateFolderName(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "fld_upd")
	folder := createTestFolder(t, q, user.ID, "Vieja")

	updated, err := q.UpdateFolderName(ctx, UpdateFolderNameParams{
		ID:         folder.ID,
		UserID:     user.ID,
		FolderName: "Nueva",
	})
	if err != nil {
		t.Fatalf("UpdateFolderName falló: %v", err)
	}
	if updated.FolderName != "Nueva" {
		t.Errorf("folder_name = %q, quería %q", updated.FolderName, "Nueva")
	}

	// Otro usuario no puede renombrarla.
	other := createTestUser(t, q, "fld_upd2")
	_, err = q.UpdateFolderName(ctx, UpdateFolderNameParams{
		ID:         folder.ID,
		UserID:     other.ID,
		FolderName: "Hackeada",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("se esperaba sql.ErrNoRows para otro usuario, se obtuvo: %v", err)
	}
}

func TestDeleteFolder(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "fld_del")
	folder := createTestFolder(t, q, user.ID, "Borrar")

	// Otro usuario no puede borrarla (la query no falla, pero no borra nada).
	other := createTestUser(t, q, "fld_del2")
	if err := q.DeleteFolder(ctx, DeleteFolderParams{ID: folder.ID, UserID: other.ID}); err != nil {
		t.Fatalf("DeleteFolder (otro usuario) falló: %v", err)
	}
	if _, err := q.GetFolderByID(ctx, GetFolderByIDParams{ID: folder.ID, UserID: user.ID}); err != nil {
		t.Fatalf("la carpeta no debería haberse borrado: %v", err)
	}

	if err := q.DeleteFolder(ctx, DeleteFolderParams{ID: folder.ID, UserID: user.ID}); err != nil {
		t.Fatalf("DeleteFolder falló: %v", err)
	}
	if _, err := q.GetFolderByID(ctx, GetFolderByIDParams{ID: folder.ID, UserID: user.ID}); err == nil {
		t.Errorf("se esperaba un error al buscar una carpeta eliminada, pero no hubo ninguno")
	}
}
