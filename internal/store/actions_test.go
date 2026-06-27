package store

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/t0mer/raptor/internal/models"
)

func TestActionCRUDAndOrdering(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	tok := newToken()
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatal(err)
	}

	// Append two actions; positions auto-increment.
	a1 := &models.Action{UUID: uuid.NewString(), TokenID: tok.UUID, Type: "set_variable",
		Parameters: map[string]any{"name": "x", "value": "1"}}
	a2 := &models.Action{UUID: uuid.NewString(), TokenID: tok.UUID, Type: "stop"}
	if err := s.CreateAction(ctx, a1); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAction(ctx, a2); err != nil {
		t.Fatal(err)
	}
	if a1.Position != 1 || a2.Position != 2 {
		t.Errorf("positions = %d, %d, want 1, 2", a1.Position, a2.Position)
	}

	list, err := s.ListActions(ctx, tok.UUID)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListActions: %v len=%d", err, len(list))
	}
	if list[0].UUID != a1.UUID || list[1].UUID != a2.UUID {
		t.Error("actions out of order")
	}
	if list[0].Parameters["name"] != "x" {
		t.Errorf("parameters lost: %+v", list[0].Parameters)
	}

	a1.Name = "set x"
	a1.Disabled = true
	if err := s.UpdateAction(ctx, a1); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetAction(ctx, a1.UUID)
	if got.Name != "set x" || !got.Disabled {
		t.Errorf("update not persisted: %+v", got)
	}

	if err := s.DeleteAction(ctx, a2.UUID); err != nil {
		t.Fatal(err)
	}
	list, _ = s.ListActions(ctx, tok.UUID)
	if len(list) != 1 {
		t.Errorf("after delete len = %d, want 1", len(list))
	}
}

func TestActionRunsLogAndCascade(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	tok := newToken()
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatal(err)
	}
	req := &models.Request{UUID: uuid.NewString(), TokenID: tok.UUID, Method: "POST"}
	if err := s.CreateRequest(ctx, req, 0); err != nil {
		t.Fatal(err)
	}

	run := &models.ActionRun{ID: uuid.NewString(), RequestID: req.UUID, ActionID: uuid.NewString(),
		ActionType: "script", ActionName: "compute", Position: 1, Output: "ok"}
	if err := s.CreateActionRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	runs, err := s.ListActionRuns(ctx, req.UUID)
	if err != nil || len(runs) != 1 || runs[0].Output != "ok" {
		t.Fatalf("ListActionRuns: %v %+v", err, runs)
	}

	// Deleting the request cascades to its action runs.
	if err := s.DeleteRequest(ctx, req.UUID); err != nil {
		t.Fatal(err)
	}
	runs, _ = s.ListActionRuns(ctx, req.UUID)
	if len(runs) != 0 {
		t.Errorf("action runs not cascade-deleted: %d", len(runs))
	}
}
