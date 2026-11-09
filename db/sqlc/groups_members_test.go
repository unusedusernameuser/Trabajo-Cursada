package db

import (
	"context"
	"testing"
)

func TestAddGroupMember(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "gm_add")
	member := createTestUser(t, q, "gm_add2")
	group := createTestGroup(t, q, owner.ID, "Grupo add")

	gm, err := q.AddGroupMember(ctx, AddGroupMemberParams{GroupID: group.ID, UserID: member.ID})
	if err != nil {
		t.Fatalf("AddGroupMember falló: %v", err)
	}
	if gm.GroupID != group.ID || gm.UserID != member.ID {
		t.Errorf("miembro = (%d, %d), quería (%d, %d)", gm.GroupID, gm.UserID, group.ID, member.ID)
	}
	if !gm.JoinedAt.Valid {
		t.Errorf("joined_at debería tener valor por defecto")
	}

	// Agregarlo dos veces viola la PK (group_id, user_id).
	if _, err := q.AddGroupMember(ctx, AddGroupMemberParams{GroupID: group.ID, UserID: member.ID}); err == nil {
		t.Errorf("se esperaba un error al agregar un miembro duplicado, pero no hubo ninguno")
	}
}

func TestRemoveGroupMember(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "gm_rm")
	member := createTestUser(t, q, "gm_rm2")
	group := createTestGroup(t, q, owner.ID, "Grupo rm")

	if _, err := q.AddGroupMember(ctx, AddGroupMemberParams{GroupID: group.ID, UserID: member.ID}); err != nil {
		t.Fatalf("AddGroupMember falló: %v", err)
	}

	if err := q.RemoveGroupMember(ctx, RemoveGroupMemberParams{GroupID: group.ID, UserID: member.ID}); err != nil {
		t.Fatalf("RemoveGroupMember falló: %v", err)
	}
	isMember, err := q.IsGroupMember(ctx, IsGroupMemberParams{GroupID: group.ID, UserID: member.ID})
	if err != nil {
		t.Fatalf("IsGroupMember falló: %v", err)
	}
	if isMember {
		t.Errorf("el miembro debería haber sido removido")
	}
}

func TestRemoveGroupMemberOwnerNotAllowed(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "gm_own")
	group := createTestGroup(t, q, owner.ID, "Grupo owner")

	// La query no permite remover al dueño del grupo (no falla, pero no borra nada).
	if err := q.RemoveGroupMember(ctx, RemoveGroupMemberParams{GroupID: group.ID, UserID: owner.ID}); err != nil {
		t.Fatalf("RemoveGroupMember falló: %v", err)
	}
	isMember, err := q.IsGroupMember(ctx, IsGroupMemberParams{GroupID: group.ID, UserID: owner.ID})
	if err != nil {
		t.Fatalf("IsGroupMember falló: %v", err)
	}
	if !isMember {
		t.Errorf("el dueño no debería poder ser removido del grupo")
	}
}

func TestIsGroupMember(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "gm_is")
	stranger := createTestUser(t, q, "gm_is2")
	group := createTestGroup(t, q, owner.ID, "Grupo is")

	isOwner, err := q.IsGroupMember(ctx, IsGroupMemberParams{GroupID: group.ID, UserID: owner.ID})
	if err != nil {
		t.Fatalf("IsGroupMember falló: %v", err)
	}
	if !isOwner {
		t.Errorf("el dueño debería ser miembro")
	}

	isStranger, err := q.IsGroupMember(ctx, IsGroupMemberParams{GroupID: group.ID, UserID: stranger.ID})
	if err != nil {
		t.Fatalf("IsGroupMember falló: %v", err)
	}
	if isStranger {
		t.Errorf("un usuario ajeno no debería ser miembro")
	}
}

func TestListGroupMembers(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "gm_ls")
	member := createTestUser(t, q, "gm_ls2")
	group := createTestGroup(t, q, owner.ID, "Grupo ls")

	if _, err := q.AddGroupMember(ctx, AddGroupMemberParams{GroupID: group.ID, UserID: member.ID}); err != nil {
		t.Fatalf("AddGroupMember falló: %v", err)
	}

	members, err := q.ListGroupMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("ListGroupMembers falló: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("len(members) = %d, quería 2", len(members))
	}

	ids := map[int32]bool{}
	for _, m := range members {
		ids[m.ID] = true
	}
	if !ids[owner.ID] || !ids[member.ID] {
		t.Errorf("faltan miembros en ListGroupMembers: %v", ids)
	}
}

func TestCountGroupMembers(t *testing.T) {
	conn := getTestDB(t)
	q := New(conn)
	ctx := context.Background()

	owner := createTestUser(t, q, "gm_cnt")
	member := createTestUser(t, q, "gm_cnt2")
	group := createTestGroup(t, q, owner.ID, "Grupo cnt")

	count, err := q.CountGroupMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("CountGroupMembers falló: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, quería 1 (solo el dueño)", count)
	}

	if _, err := q.AddGroupMember(ctx, AddGroupMemberParams{GroupID: group.ID, UserID: member.ID}); err != nil {
		t.Fatalf("AddGroupMember falló: %v", err)
	}
	count, err = q.CountGroupMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("CountGroupMembers falló: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, quería 2", count)
	}
}
