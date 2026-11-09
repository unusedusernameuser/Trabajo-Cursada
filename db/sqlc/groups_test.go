package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestCreateAndGetGroup(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "grp_cg")

	created, err := q.CreateGroup(ctx, CreateGroupParams{
		OwnerID:          owner.ID,
		GroupName:        "Equipo",
		GroupDescription: sql.NullString{String: "Grupo de prueba", Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateGroup falló: %v", err)
	}
	if created.GroupName != "Equipo" {
		t.Errorf("group_name = %q, quería %q", created.GroupName, "Equipo")
	}
	if created.OwnerID != owner.ID {
		t.Errorf("owner_id = %d, quería %d", created.OwnerID, owner.ID)
	}
	if !created.GroupDescription.Valid || created.GroupDescription.String != "Grupo de prueba" {
		t.Errorf("group_description = %+v, quería %q", created.GroupDescription, "Grupo de prueba")
	}

	got, err := q.GetGroupByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetGroupByID falló: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetGroupByID: id = %d, quería %d", got.ID, created.ID)
	}

	// CreateGroup agrega automáticamente al dueño como miembro.
	isMember, err := q.IsGroupMember(ctx, IsGroupMemberParams{GroupID: created.ID, UserID: owner.ID})
	if err != nil {
		t.Fatalf("IsGroupMember falló: %v", err)
	}
	if !isMember {
		t.Errorf("el dueño debería ser miembro del grupo al crearlo")
	}
}

func TestCreateGroupWithoutDescription(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "grp_nodesc")

	created, err := q.CreateGroup(ctx, CreateGroupParams{OwnerID: owner.ID, GroupName: "Sin desc"})
	if err != nil {
		t.Fatalf("CreateGroup falló: %v", err)
	}
	if created.GroupDescription.Valid {
		t.Errorf("group_description debería ser NULL, fue %+v", created.GroupDescription)
	}
}

func TestListGroupsByOwner(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "grp_lo")
	other := createTestUser(t, q, "grp_lo2")

	createTestGroup(t, q, owner.ID, "B grupo")
	createTestGroup(t, q, owner.ID, "A grupo")
	createTestGroup(t, q, other.ID, "Ajeno")

	groups, err := q.ListGroupsByOwner(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListGroupsByOwner falló: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, quería 2", len(groups))
	}
	// ORDER BY group_name
	if groups[0].GroupName != "A grupo" || groups[1].GroupName != "B grupo" {
		t.Errorf("orden incorrecto: %q, %q", groups[0].GroupName, groups[1].GroupName)
	}
}

func TestListGroupsByMember(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "grp_lm")
	member := createTestUser(t, q, "grp_lm2")

	joined := createTestGroup(t, q, owner.ID, "Al que me uno")
	createTestGroup(t, q, owner.ID, "Al que no me uno")

	if _, err := q.AddGroupMember(ctx, AddGroupMemberParams{GroupID: joined.ID, UserID: member.ID}); err != nil {
		t.Fatalf("AddGroupMember falló: %v", err)
	}

	groups, err := q.ListGroupsByMember(ctx, member.ID)
	if err != nil {
		t.Fatalf("ListGroupsByMember falló: %v", err)
	}
	if len(groups) != 1 || groups[0].ID != joined.ID {
		t.Errorf("ListGroupsByMember = %+v, se esperaba solo el grupo %d", groups, joined.ID)
	}

	// El dueño es miembro de ambos grupos.
	ownerGroups, err := q.ListGroupsByMember(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListGroupsByMember (dueño) falló: %v", err)
	}
	if len(ownerGroups) != 2 {
		t.Errorf("len(ownerGroups) = %d, quería 2", len(ownerGroups))
	}
}

func TestUpdateGroup(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "grp_upd")
	group := createTestGroup(t, q, owner.ID, "Viejo")

	updated, err := q.UpdateGroup(ctx, UpdateGroupParams{
		ID:               group.ID,
		OwnerID:          owner.ID,
		GroupName:        "Nuevo",
		GroupDescription: sql.NullString{},
	})
	if err != nil {
		t.Fatalf("UpdateGroup falló: %v", err)
	}
	if updated.GroupName != "Nuevo" {
		t.Errorf("group_name = %q, quería %q", updated.GroupName, "Nuevo")
	}
	if updated.GroupDescription.Valid {
		t.Errorf("group_description debería ser NULL, fue %+v", updated.GroupDescription)
	}

	// Solo el dueño puede actualizarlo.
	other := createTestUser(t, q, "grp_upd2")
	_, err = q.UpdateGroup(ctx, UpdateGroupParams{
		ID:        group.ID,
		OwnerID:   other.ID,
		GroupName: "Hackeado",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("se esperaba sql.ErrNoRows para un no-dueño, se obtuvo: %v", err)
	}
}

func TestDeleteGroup(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "grp_del")
	group := createTestGroup(t, q, owner.ID, "Borrar")

	// Un no-dueño no puede borrarlo.
	other := createTestUser(t, q, "grp_del2")
	if err := q.DeleteGroup(ctx, DeleteGroupParams{ID: group.ID, OwnerID: other.ID}); err != nil {
		t.Fatalf("DeleteGroup (no-dueño) falló: %v", err)
	}
	if _, err := q.GetGroupByID(ctx, group.ID); err != nil {
		t.Fatalf("el grupo no debería haberse borrado: %v", err)
	}

	if err := q.DeleteGroup(ctx, DeleteGroupParams{ID: group.ID, OwnerID: owner.ID}); err != nil {
		t.Fatalf("DeleteGroup falló: %v", err)
	}
	if _, err := q.GetGroupByID(ctx, group.ID); err == nil {
		t.Errorf("se esperaba un error al buscar un grupo eliminado, pero no hubo ninguno")
	}
}
