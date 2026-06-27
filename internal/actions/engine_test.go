package actions

import (
	"context"
	"testing"

	"github.com/t0mer/raptor/internal/models"
)

func testToken() *models.Token {
	return &models.Token{UUID: "tok", DefaultStatus: 200, DefaultContent: "default", DefaultContentType: "text/plain"}
}

func act(typ string, params map[string]any) *models.Action {
	return &models.Action{UUID: typ, TokenID: "tok", Type: typ, Parameters: params}
}

func TestInterpolation(t *testing.T) {
	req := &models.Request{
		Content: "hello",
		Method:  "POST",
		Query:   map[string][]string{"id": {"42"}},
		Headers: map[string][]string{"X-Token": {"secret"}},
	}
	ec := &ExecContext{Request: req, Vars: map[string]string{"name": "Alice"}}
	cases := map[string]string{
		"hi $name$":                "hi Alice",
		"$request.method$":         "POST",
		"$request.content$":        "hello",
		"id=$request.query.id$":    "id=42",
		"$request.header.x-token$": "secret",
		"$missing$":                "",
	}
	for in, want := range cases {
		if got := ec.Interp(in); got != want {
			t.Errorf("Interp(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestChainSetVarAndModifyResponse(t *testing.T) {
	e := New()
	req := &models.Request{Content: "ping"}
	ec := e.NewContext(req, testToken())

	acts := []*models.Action{
		act("set_variable", map[string]any{"name": "greeting", "value": "echo: $request.content$"}),
		act("modify_response", map[string]any{"status": "201", "content": "$greeting$", "content_type": "text/plain"}),
	}
	results := e.Execute(context.Background(), acts, ec)
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	for _, r := range results {
		if r.Err != nil {
			t.Fatalf("action %s error: %v", r.Action.Type, r.Err)
		}
	}
	if ec.Vars["greeting"] != "echo: ping" {
		t.Errorf("greeting = %q", ec.Vars["greeting"])
	}
	if ec.Response.Status != 201 || ec.Response.Content != "echo: ping" {
		t.Errorf("response = %d %q", ec.Response.Status, ec.Response.Content)
	}
	if !ec.Response.Set {
		t.Error("Response.Set should be true")
	}
}

func TestStopHaltsChain(t *testing.T) {
	e := New()
	ec := e.NewContext(&models.Request{}, testToken())
	acts := []*models.Action{
		act("stop", nil),
		act("set_variable", map[string]any{"name": "x", "value": "1"}),
	}
	results := e.Execute(context.Background(), acts, ec)
	if len(results) != 1 {
		t.Fatalf("chain did not stop: %d results", len(results))
	}
	if _, ok := ec.Vars["x"]; ok {
		t.Error("action after stop should not have run")
	}
}

func TestDisabledSkipped(t *testing.T) {
	e := New()
	ec := e.NewContext(&models.Request{}, testToken())
	a := act("set_variable", map[string]any{"name": "x", "value": "1"})
	a.Disabled = true
	results := e.Execute(context.Background(), []*models.Action{a}, ec)
	if len(results) != 0 {
		t.Errorf("disabled action ran: %d results", len(results))
	}
}

func TestDontSave(t *testing.T) {
	e := New()
	ec := e.NewContext(&models.Request{}, testToken())
	e.Execute(context.Background(), []*models.Action{act("dont_save", nil)}, ec)
	if !ec.DontSave {
		t.Error("DontSave not set")
	}
}
