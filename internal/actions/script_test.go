package actions

import (
	"context"
	"testing"

	"github.com/t0mer/raptor/internal/models"
)

func TestScriptRespondAndVars(t *testing.T) {
	e := New()
	ec := e.NewContext(&models.Request{Content: `{"amount": 42}`, Method: "POST"}, testToken())
	src := `
		var data = JSON.parse(request.content);
		set("amount", String(data.amount));
		echo("amount is", data.amount);
		if (data.amount > 10) {
			respond('{"status":"big"}', 201, "application/json");
		}
	`
	res := e.Execute(context.Background(), []*models.Action{act("script", map[string]any{"script": src})}, ec)
	if res[0].Err != nil {
		t.Fatalf("script error: %v", res[0].Err)
	}
	if ec.Vars["amount"] != "42" {
		t.Errorf("amount var = %q", ec.Vars["amount"])
	}
	if ec.Response.Status != 201 || ec.Response.Content != `{"status":"big"}` {
		t.Errorf("response = %d %q", ec.Response.Status, ec.Response.Content)
	}
	if res[0].Output == "" {
		t.Error("echo output not captured")
	}
}

func TestScriptStopAndDontSave(t *testing.T) {
	e := New()
	ec := e.NewContext(&models.Request{}, testToken())
	e.Execute(context.Background(), []*models.Action{
		act("script", map[string]any{"script": `dont_save(); stop();`}),
		act("set_variable", map[string]any{"name": "x", "value": "1"}),
	}, ec)
	if !ec.DontSave || !ec.Stopped {
		t.Errorf("dont_save=%v stopped=%v", ec.DontSave, ec.Stopped)
	}
	if _, ok := ec.Vars["x"]; ok {
		t.Error("action after script stop ran")
	}
}

func TestScriptError(t *testing.T) {
	e := New()
	ec := e.NewContext(&models.Request{}, testToken())
	res := e.Execute(context.Background(), []*models.Action{
		act("script", map[string]any{"script": `throw new Error("boom")`}),
	}, ec)
	if res[0].Err == nil {
		t.Error("expected script error")
	}
}
