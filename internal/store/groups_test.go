package store

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/models"
)

func TestGroupCRUDAndTokenDetach(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	g := &models.Group{ID: uuid.NewString(), Name: "Payments", Color: "#4f46e5"}
	if err := s.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	got, err := s.GetGroup(ctx, g.ID)
	if err != nil || got.Name != "Payments" || got.Color != "#4f46e5" {
		t.Fatalf("GetGroup: %v / %+v", err, got)
	}

	g.Name = "Billing"
	if err := s.UpdateGroup(ctx, g); err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	got, _ = s.GetGroup(ctx, g.ID)
	if got.Name != "Billing" {
		t.Errorf("name = %q, want Billing", got.Name)
	}

	list, err := s.ListGroups(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListGroups: %v len=%d", err, len(list))
	}

	// A token in the group should be detached (not deleted) on group delete.
	tok := newToken()
	tok.GroupID = g.ID
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteGroup(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if _, err := s.GetGroup(ctx, g.ID); err != ErrNotFound {
		t.Errorf("group still present: %v", err)
	}
	reloaded, err := s.GetToken(ctx, tok.UUID)
	if err != nil {
		t.Fatalf("token gone after group delete: %v", err)
	}
	if reloaded.GroupID != "" {
		t.Errorf("token group_id = %q, want cleared", reloaded.GroupID)
	}
}
