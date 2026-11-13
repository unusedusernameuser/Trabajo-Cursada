package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestCreateAndGetTask(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tsk_cg")
	folder := createTestFolder(t, q, user.ID, "Carpeta")
	due := time.Now().Add(48 * time.Hour)

	created, err := q.CreateTask(ctx, CreateTaskParams{
		UserID:          user.ID,
		FolderID:        sql.NullInt32{Int32: folder.ID, Valid: true},
		Title:           "Mi tarea",
		TaskDescription: sql.NullString{String: "detalle", Valid: true},
		DueDate:         sql.NullTime{Time: due, Valid: true},
		TaskPriority:    sql.NullString{String: "alta", Valid: true},
		AttachmentPath:  sql.NullString{String: "/tmp/archivo.pdf", Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateTask falló: %v", err)
	}
	if created.Title != "Mi tarea" {
		t.Errorf("title = %q, quería %q", created.Title, "Mi tarea")
	}
	if created.TaskPriority != "alta" {
		t.Errorf("task_priority = %q, quería %q", created.TaskPriority, "alta")
	}
	if created.TaskStatus != "pendiente" {
		t.Errorf("task_status = %q, quería %q", created.TaskStatus, "pendiente")
	}
	if !created.FolderID.Valid || created.FolderID.Int32 != folder.ID {
		t.Errorf("folder_id = %+v, quería %d", created.FolderID, folder.ID)
	}
	if created.CompletedAt.Valid {
		t.Errorf("completed_at debería ser NULL")
	}

	got, err := q.GetTaskByID(ctx, GetTaskByIDParams{ID: created.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("GetTaskByID falló: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetTaskByID: id = %d, quería %d", got.ID, created.ID)
	}
}

func TestCreateTaskDefaultPriority(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)

	user := createTestUser(t, q, "tsk_prio")
	task := createTestTask(t, q, user.ID, "Sin prioridad")

	if task.TaskPriority != "media" {
		t.Errorf("task_priority = %q, quería el valor por defecto %q", task.TaskPriority, "media")
	}
}

func TestCreateTaskValidations(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tsk_val")
	other := createTestUser(t, q, "tsk_val2")
	foreignFolder := createTestFolder(t, q, other.ID, "Ajena")

	// Carpeta de otro usuario: no inserta.
	_, err := q.CreateTask(ctx, CreateTaskParams{
		UserID:   user.ID,
		FolderID: sql.NullInt32{Int32: foreignFolder.ID, Valid: true},
		Title:    "Carpeta ajena",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("carpeta ajena: se esperaba sql.ErrNoRows, se obtuvo: %v", err)
	}
}

func TestListTasksByUser(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tsk_lu")
	other := createTestUser(t, q, "tsk_lu2")
	folder := createTestFolder(t, q, user.ID, "Carpeta")

	low, err := q.CreateTask(ctx, CreateTaskParams{
		UserID:       user.ID,
		FolderID:     sql.NullInt32{Int32: folder.ID, Valid: true},
		Title:        "Baja",
		TaskPriority: sql.NullString{String: "baja", Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateTask falló: %v", err)
	}
	high := createTestTask(t, q, user.ID, "Alta")
	if _, err := q.UpdateTask(ctx, UpdateTaskParams{
		ID: high.ID, UserID: user.ID, Title: "Alta", TaskPriority: "alta",
	}); err != nil {
		t.Fatalf("UpdateTask falló: %v", err)
	}
	if _, err := q.UpdateTaskStatus(ctx, UpdateTaskStatusParams{
		ID: high.ID, UserID: user.ID, TaskStatus: "en_progreso",
	}); err != nil {
		t.Fatalf("UpdateTaskStatus falló: %v", err)
	}
	createTestTask(t, q, other.ID, "De otro")

	// Sin filtros.
	all, err := q.ListTasksByUser(ctx, ListTasksByUserParams{UserID: user.ID})
	if err != nil {
		t.Fatalf("ListTasksByUser falló: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("sin filtros: len = %d, quería 2", len(all))
	}

	// Filtro por estado.
	byStatus, err := q.ListTasksByUser(ctx, ListTasksByUserParams{
		UserID:     user.ID,
		TaskStatus: sql.NullString{String: "en_progreso", Valid: true},
	})
	if err != nil {
		t.Fatalf("ListTasksByUser (estado) falló: %v", err)
	}
	if len(byStatus) != 1 || byStatus[0].ID != high.ID {
		t.Errorf("filtro por estado = %+v, se esperaba solo la tarea %d", byStatus, high.ID)
	}

	// Filtro por prioridad.
	byPriority, err := q.ListTasksByUser(ctx, ListTasksByUserParams{
		UserID:       user.ID,
		TaskPriority: sql.NullString{String: "baja", Valid: true},
	})
	if err != nil {
		t.Fatalf("ListTasksByUser (prioridad) falló: %v", err)
	}
	if len(byPriority) != 1 || byPriority[0].ID != low.ID {
		t.Errorf("filtro por prioridad = %+v, se esperaba solo la tarea %d", byPriority, low.ID)
	}

	// Filtro por carpeta.
	byFolder, err := q.ListTasksByUser(ctx, ListTasksByUserParams{
		UserID:   user.ID,
		FolderID: sql.NullInt32{Int32: folder.ID, Valid: true},
	})
	if err != nil {
		t.Fatalf("ListTasksByUser (carpeta) falló: %v", err)
	}
	if len(byFolder) != 1 || byFolder[0].ID != low.ID {
		t.Errorf("filtro por carpeta = %+v, se esperaba solo la tarea %d", byFolder, low.ID)
	}
}

func TestListTasksByFolder(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tsk_lf")
	folder := createTestFolder(t, q, user.ID, "Carpeta")

	inFolder, err := q.CreateTask(ctx, CreateTaskParams{
		UserID:   user.ID,
		FolderID: sql.NullInt32{Int32: folder.ID, Valid: true},
		Title:    "En carpeta",
	})
	if err != nil {
		t.Fatalf("CreateTask falló: %v", err)
	}
	createTestTask(t, q, user.ID, "Fuera de carpeta")

	tasks, err := q.ListTasksByFolder(ctx, ListTasksByFolderParams{
		FolderID: sql.NullInt32{Int32: folder.ID, Valid: true},
		UserID:   user.ID,
	})
	if err != nil {
		t.Fatalf("ListTasksByFolder falló: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != inFolder.ID {
		t.Errorf("tasks = %+v, se esperaba solo la tarea %d", tasks, inFolder.ID)
	}
}

func TestListOverdueTasksByUser(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tsk_od")
	past := sql.NullTime{Time: time.Now().Add(-24 * time.Hour), Valid: true}
	future := sql.NullTime{Time: time.Now().Add(24 * time.Hour), Valid: true}

	overdue, err := q.CreateTask(ctx, CreateTaskParams{UserID: user.ID, Title: "Vencida", DueDate: past})
	if err != nil {
		t.Fatalf("CreateTask falló: %v", err)
	}
	if _, err := q.CreateTask(ctx, CreateTaskParams{UserID: user.ID, Title: "Futura", DueDate: future}); err != nil {
		t.Fatalf("CreateTask falló: %v", err)
	}
	done, err := q.CreateTask(ctx, CreateTaskParams{UserID: user.ID, Title: "Vencida pero completada", DueDate: past})
	if err != nil {
		t.Fatalf("CreateTask falló: %v", err)
	}
	if _, err := q.UpdateTaskStatus(ctx, UpdateTaskStatusParams{
		ID: done.ID, UserID: user.ID, TaskStatus: "completada",
	}); err != nil {
		t.Fatalf("UpdateTaskStatus falló: %v", err)
	}

	tasks, err := q.ListOverdueTasksByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListOverdueTasksByUser falló: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != overdue.ID {
		t.Errorf("tasks = %+v, se esperaba solo la tarea %d", tasks, overdue.ID)
	}
}

func TestUpdateTask(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tsk_upd")
	task := createTestTask(t, q, user.ID, "Original")
	due := time.Now().Add(72 * time.Hour)

	updated, err := q.UpdateTask(ctx, UpdateTaskParams{
		ID:              task.ID,
		UserID:          user.ID,
		Title:           "Modificada",
		TaskDescription: sql.NullString{String: "nueva descripción", Valid: true},
		DueDate:         sql.NullTime{Time: due, Valid: true},
		TaskPriority:    "baja",
	})
	if err != nil {
		t.Fatalf("UpdateTask falló: %v", err)
	}
	if updated.Title != "Modificada" {
		t.Errorf("title = %q, quería %q", updated.Title, "Modificada")
	}
	if updated.TaskPriority != "baja" {
		t.Errorf("task_priority = %q, quería %q", updated.TaskPriority, "baja")
	}
	if !updated.TaskDescription.Valid || updated.TaskDescription.String != "nueva descripción" {
		t.Errorf("task_description = %+v", updated.TaskDescription)
	}
	if !updated.DueDate.Valid {
		t.Errorf("due_date debería tener valor")
	}

	// Solo el dueño puede modificarla.
	other := createTestUser(t, q, "tsk_upd2")
	_, err = q.UpdateTask(ctx, UpdateTaskParams{
		ID: task.ID, UserID: other.ID, Title: "Hackeada", TaskPriority: "alta",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("se esperaba sql.ErrNoRows para otro usuario, se obtuvo: %v", err)
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "tsk_st")
	stranger := createTestUser(t, q, "tsk_st2")
	task := createTestTask(t, q, owner.ID, "Estado")

	// Al completarla, completed_at se completa.
	completed, err := q.UpdateTaskStatus(ctx, UpdateTaskStatusParams{
		ID: task.ID, UserID: owner.ID, TaskStatus: "completada",
	})
	if err != nil {
		t.Fatalf("UpdateTaskStatus falló: %v", err)
	}
	if completed.TaskStatus != "completada" {
		t.Errorf("task_status = %q, quería %q", completed.TaskStatus, "completada")
	}
	if !completed.CompletedAt.Valid {
		t.Errorf("completed_at debería tener valor al completar la tarea")
	}

	// Al volver a otro estado, completed_at vuelve a NULL.
	reopened, err := q.UpdateTaskStatus(ctx, UpdateTaskStatusParams{
		ID: task.ID, UserID: owner.ID, TaskStatus: "en_progreso",
	})
	if err != nil {
		t.Fatalf("UpdateTaskStatus (reabrir) falló: %v", err)
	}
	if reopened.CompletedAt.Valid {
		t.Errorf("completed_at debería ser NULL al reabrir la tarea")
	}

	// Otro usuario no puede cambiarlo.
	_, err = q.UpdateTaskStatus(ctx, UpdateTaskStatusParams{
		ID: task.ID, UserID: stranger.ID, TaskStatus: "completada",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("se esperaba sql.ErrNoRows para otro usuario, se obtuvo: %v", err)
	}
}

func TestMoveTaskToFolder(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tsk_mv")
	other := createTestUser(t, q, "tsk_mv2")
	folder := createTestFolder(t, q, user.ID, "Destino")
	foreignFolder := createTestFolder(t, q, other.ID, "Ajena")
	task := createTestTask(t, q, user.ID, "Mover")

	moved, err := q.MoveTaskToFolder(ctx, MoveTaskToFolderParams{
		ID:       task.ID,
		UserID:   user.ID,
		FolderID: sql.NullInt32{Int32: folder.ID, Valid: true},
	})
	if err != nil {
		t.Fatalf("MoveTaskToFolder falló: %v", err)
	}
	if !moved.FolderID.Valid || moved.FolderID.Int32 != folder.ID {
		t.Errorf("folder_id = %+v, quería %d", moved.FolderID, folder.ID)
	}

	// Carpeta de otro usuario: no actualiza.
	_, err = q.MoveTaskToFolder(ctx, MoveTaskToFolderParams{
		ID:       task.ID,
		UserID:   user.ID,
		FolderID: sql.NullInt32{Int32: foreignFolder.ID, Valid: true},
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("carpeta ajena: se esperaba sql.ErrNoRows, se obtuvo: %v", err)
	}

	// Sacarla de la carpeta (NULL).
	removed, err := q.MoveTaskToFolder(ctx, MoveTaskToFolderParams{ID: task.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("MoveTaskToFolder (NULL) falló: %v", err)
	}
	if removed.FolderID.Valid {
		t.Errorf("folder_id debería ser NULL, fue %+v", removed.FolderID)
	}
}

func TestUpdateTaskAttachment(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tsk_att")
	task := createTestTask(t, q, user.ID, "Adjunto")

	updated, err := q.UpdateTaskAttachment(ctx, UpdateTaskAttachmentParams{
		ID:             task.ID,
		UserID:         user.ID,
		AttachmentPath: sql.NullString{String: "/uploads/doc.pdf", Valid: true},
	})
	if err != nil {
		t.Fatalf("UpdateTaskAttachment falló: %v", err)
	}
	if !updated.AttachmentPath.Valid || updated.AttachmentPath.String != "/uploads/doc.pdf" {
		t.Errorf("attachment_path = %+v, quería %q", updated.AttachmentPath, "/uploads/doc.pdf")
	}

	// Quitar el adjunto (NULL).
	cleared, err := q.UpdateTaskAttachment(ctx, UpdateTaskAttachmentParams{ID: task.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("UpdateTaskAttachment (NULL) falló: %v", err)
	}
	if cleared.AttachmentPath.Valid {
		t.Errorf("attachment_path debería ser NULL, fue %+v", cleared.AttachmentPath)
	}

	other := createTestUser(t, q, "tsk_att2")
	_, err = q.UpdateTaskAttachment(ctx, UpdateTaskAttachmentParams{
		ID:             task.ID,
		UserID:         other.ID,
		AttachmentPath: sql.NullString{String: "/x.pdf", Valid: true},
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("se esperaba sql.ErrNoRows para otro usuario, se obtuvo: %v", err)
	}
}

func TestDeleteTask(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	user := createTestUser(t, q, "tsk_del")
	other := createTestUser(t, q, "tsk_del2")
	task := createTestTask(t, q, user.ID, "Borrar")

	// Otro usuario no puede borrarla.
	if err := q.DeleteTask(ctx, DeleteTaskParams{ID: task.ID, UserID: other.ID}); err != nil {
		t.Fatalf("DeleteTask (otro usuario) falló: %v", err)
	}
	if _, err := q.GetTaskByID(ctx, GetTaskByIDParams{ID: task.ID, UserID: user.ID}); err != nil {
		t.Fatalf("la tarea no debería haberse borrado: %v", err)
	}

	if err := q.DeleteTask(ctx, DeleteTaskParams{ID: task.ID, UserID: user.ID}); err != nil {
		t.Fatalf("DeleteTask falló: %v", err)
	}
	if _, err := q.GetTaskByID(ctx, GetTaskByIDParams{ID: task.ID, UserID: user.ID}); err == nil {
		t.Errorf("se esperaba un error al buscar una tarea eliminada, pero no hubo ninguno")
	}
}
