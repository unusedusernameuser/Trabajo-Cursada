package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestCreateAndGetTag(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tag_cg")

	created, err := q.CreateTag(ctx, CreateTagParams{UserID: user.ID, TagName: "urgente"})
	if err != nil {
		t.Fatalf("CreateTag falló: %v", err)
	}
	if created.TagName != "urgente" {
		t.Errorf("tag_name = %q, quería %q", created.TagName, "urgente")
	}

	got, err := q.GetTagByID(ctx, GetTagByIDParams{ID: created.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("GetTagByID falló: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetTagByID: id = %d, quería %d", got.ID, created.ID)
	}

	other := createTestUser(t, q, "tag_cg2")
	_, err = q.GetTagByID(ctx, GetTagByIDParams{ID: created.ID, UserID: other.ID})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("se esperaba sql.ErrNoRows para otro usuario, se obtuvo: %v", err)
	}
}

func TestCreateTagDuplicate(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tag_dup")
	createTestTag(t, q, user.ID, "repetido")

	_, err := q.CreateTag(ctx, CreateTagParams{UserID: user.ID, TagName: "repetido"})
	if err == nil {
		t.Errorf("se esperaba un error por (user_id, tag_name) duplicado, pero no hubo ninguno")
	}
}

func TestListTagsByUser(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tag_list")
	other := createTestUser(t, q, "tag_list2")

	createTestTag(t, q, user.ID, "b_tag")
	createTestTag(t, q, user.ID, "a_tag")
	createTestTag(t, q, other.ID, "ajeno")

	tags, err := q.ListTagsByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListTagsByUser falló: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, quería 2", len(tags))
	}
	// ORDER BY tag_name
	if tags[0].TagName != "a_tag" || tags[1].TagName != "b_tag" {
		t.Errorf("orden incorrecto: %q, %q", tags[0].TagName, tags[1].TagName)
	}
}

func TestUpdateTagName(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tag_upd")
	tag := createTestTag(t, q, user.ID, "viejo")

	updated, err := q.UpdateTagName(ctx, UpdateTagNameParams{
		ID:      tag.ID,
		UserID:  user.ID,
		TagName: "nuevo",
	})
	if err != nil {
		t.Fatalf("UpdateTagName falló: %v", err)
	}
	if updated.TagName != "nuevo" {
		t.Errorf("tag_name = %q, quería %q", updated.TagName, "nuevo")
	}

	other := createTestUser(t, q, "tag_upd2")
	_, err = q.UpdateTagName(ctx, UpdateTagNameParams{
		ID:      tag.ID,
		UserID:  other.ID,
		TagName: "hackeado",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("se esperaba sql.ErrNoRows para otro usuario, se obtuvo: %v", err)
	}
}

func TestDeleteTag(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tag_del")
	tag := createTestTag(t, q, user.ID, "borrar")

	// Otro usuario no puede borrarlo.
	other := createTestUser(t, q, "tag_del2")
	if err := q.DeleteTag(ctx, DeleteTagParams{ID: tag.ID, UserID: other.ID}); err != nil {
		t.Fatalf("DeleteTag (otro usuario) falló: %v", err)
	}
	if _, err := q.GetTagByID(ctx, GetTagByIDParams{ID: tag.ID, UserID: user.ID}); err != nil {
		t.Fatalf("el tag no debería haberse borrado: %v", err)
	}

	if err := q.DeleteTag(ctx, DeleteTagParams{ID: tag.ID, UserID: user.ID}); err != nil {
		t.Fatalf("DeleteTag falló: %v", err)
	}
	if _, err := q.GetTagByID(ctx, GetTagByIDParams{ID: tag.ID, UserID: user.ID}); err == nil {
		t.Errorf("se esperaba un error al buscar un tag eliminado, pero no hubo ninguno")
	}
}
