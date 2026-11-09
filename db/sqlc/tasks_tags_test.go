package db

import (
	"context"
	"testing"
)

func TestAddTagToTaskAndListTagsByTask(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tt_add")
	task := createTestTask(t, q, user.ID, "Tarea con tags")
	tagB := createTestTag(t, q, user.ID, "b_tag")
	tagA := createTestTag(t, q, user.ID, "a_tag")

	for _, tag := range []Tag{tagB, tagA} {
		if err := q.AddTagToTask(ctx, AddTagToTaskParams{TaskID: task.ID, TagID: tag.ID, UserID: user.ID}); err != nil {
			t.Fatalf("AddTagToTask falló: %v", err)
		}
	}

	// Agregar el mismo tag otra vez no falla (ON CONFLICT DO NOTHING).
	if err := q.AddTagToTask(ctx, AddTagToTaskParams{TaskID: task.ID, TagID: tagA.ID, UserID: user.ID}); err != nil {
		t.Fatalf("AddTagToTask (duplicado) falló: %v", err)
	}

	tags, err := q.ListTagsByTask(ctx, ListTagsByTaskParams{TaskID: task.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("ListTagsByTask falló: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, quería 2", len(tags))
	}
	// ORDER BY tag_name
	if tags[0].TagName != "a_tag" || tags[1].TagName != "b_tag" {
		t.Errorf("orden incorrecto: %q, %q", tags[0].TagName, tags[1].TagName)
	}
}

func TestAddTagToTaskOtherUser(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tt_oth")
	other := createTestUser(t, q, "tt_oth2")
	task := createTestTask(t, q, user.ID, "Tarea propia")
	foreignTag := createTestTag(t, q, other.ID, "ajeno")

	// Un tag de otro usuario no se puede asociar a la tarea.
	if err := q.AddTagToTask(ctx, AddTagToTaskParams{TaskID: task.ID, TagID: foreignTag.ID, UserID: user.ID}); err != nil {
		t.Fatalf("AddTagToTask falló: %v", err)
	}
	tags, err := q.ListTagsByTask(ctx, ListTagsByTaskParams{TaskID: task.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("ListTagsByTask falló: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, quería 0 (el tag es de otro usuario)", len(tags))
	}

	// Otro usuario tampoco puede etiquetar la tarea ajena, ni siquiera con su propio tag.
	if err := q.AddTagToTask(ctx, AddTagToTaskParams{TaskID: task.ID, TagID: foreignTag.ID, UserID: other.ID}); err != nil {
		t.Fatalf("AddTagToTask falló: %v", err)
	}
	tags, err = q.ListTagsByTask(ctx, ListTagsByTaskParams{TaskID: task.ID, UserID: other.ID})
	if err != nil {
		t.Fatalf("ListTagsByTask falló: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, quería 0 (la tarea es de otro usuario)", len(tags))
	}
}

func TestRemoveTagFromTask(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tt_rm")
	task := createTestTask(t, q, user.ID, "Tarea rm")
	tag1 := createTestTag(t, q, user.ID, "tag1")
	tag2 := createTestTag(t, q, user.ID, "tag2")

	for _, tag := range []Tag{tag1, tag2} {
		if err := q.AddTagToTask(ctx, AddTagToTaskParams{TaskID: task.ID, TagID: tag.ID, UserID: user.ID}); err != nil {
			t.Fatalf("AddTagToTask falló: %v", err)
		}
	}

	if err := q.RemoveTagFromTask(ctx, RemoveTagFromTaskParams{TaskID: task.ID, TagID: tag1.ID, UserID: user.ID}); err != nil {
		t.Fatalf("RemoveTagFromTask falló: %v", err)
	}

	tags, err := q.ListTagsByTask(ctx, ListTagsByTaskParams{TaskID: task.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("ListTagsByTask falló: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != tag2.ID {
		t.Errorf("tags = %+v, se esperaba solo el tag %d", tags, tag2.ID)
	}
}

func TestClearTaskTags(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tt_clr")
	task := createTestTask(t, q, user.ID, "Tarea clear")
	tag1 := createTestTag(t, q, user.ID, "tag1")
	tag2 := createTestTag(t, q, user.ID, "tag2")

	for _, tag := range []Tag{tag1, tag2} {
		if err := q.AddTagToTask(ctx, AddTagToTaskParams{TaskID: task.ID, TagID: tag.ID, UserID: user.ID}); err != nil {
			t.Fatalf("AddTagToTask falló: %v", err)
		}
	}

	// Otro usuario no puede limpiar los tags de una tarea ajena.
	other := createTestUser(t, q, "tt_clr2")
	if err := q.ClearTaskTags(ctx, ClearTaskTagsParams{TaskID: task.ID, UserID: other.ID}); err != nil {
		t.Fatalf("ClearTaskTags (otro usuario) falló: %v", err)
	}
	tags, err := q.ListTagsByTask(ctx, ListTagsByTaskParams{TaskID: task.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("ListTagsByTask falló: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, quería 2 (no debería haberse borrado nada)", len(tags))
	}

	if err := q.ClearTaskTags(ctx, ClearTaskTagsParams{TaskID: task.ID, UserID: user.ID}); err != nil {
		t.Fatalf("ClearTaskTags falló: %v", err)
	}
	tags, err = q.ListTagsByTask(ctx, ListTagsByTaskParams{TaskID: task.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("ListTagsByTask falló: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, quería 0", len(tags))
	}
}

func TestListTasksByTag(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tt_lt")
	tag := createTestTag(t, q, user.ID, "importante")
	tagged := createTestTask(t, q, user.ID, "Con tag")
	createTestTask(t, q, user.ID, "Sin tag")

	if err := q.AddTagToTask(ctx, AddTagToTaskParams{TaskID: tagged.ID, TagID: tag.ID, UserID: user.ID}); err != nil {
		t.Fatalf("AddTagToTask falló: %v", err)
	}

	tasks, err := q.ListTasksByTag(ctx, ListTasksByTagParams{TagID: tag.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("ListTasksByTag falló: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != tagged.ID {
		t.Errorf("tasks = %+v, se esperaba solo la tarea %d", tasks, tagged.ID)
	}

	// Otro usuario no ve las tareas de un tag ajeno.
	other := createTestUser(t, q, "tt_lt2")
	tasks, err = q.ListTasksByTag(ctx, ListTasksByTagParams{TagID: tag.ID, UserID: other.ID})
	if err != nil {
		t.Fatalf("ListTasksByTag (otro usuario) falló: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("len(tasks) = %d, quería 0", len(tasks))
	}
}
