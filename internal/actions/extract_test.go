package actions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/t0mer/raptor/internal/models"
)

func TestExtractJSON(t *testing.T) {
	e := New()
	ec := e.NewContext(&models.Request{Content: `{"user":{"id":42,"name":"Bob"}}`}, testToken())
	e.Execute(context.Background(), []*models.Action{
		act("extract_json", map[string]any{"path": "user.name", "variable": "uname"}),
	}, ec)
	if ec.Vars["uname"] != "Bob" {
		t.Errorf("uname = %q, want Bob", ec.Vars["uname"])
	}
}

func TestExtractRegex(t *testing.T) {
	e := New()
	ec := e.NewContext(&models.Request{Content: "order #12345 confirmed"}, testToken())
	e.Execute(context.Background(), []*models.Action{
		act("extract_regex", map[string]any{"pattern": `#(\d+)`, "group": 1, "variable": "order"}),
	}, ec)
	if ec.Vars["order"] != "12345" {
		t.Errorf("order = %q, want 12345", ec.Vars["order"])
	}
}

func TestConditionsStopAndSkip(t *testing.T) {
	e := New()

	// equals → stop halts the chain
	ec := e.NewContext(&models.Request{Method: "DELETE"}, testToken())
	res := e.Execute(context.Background(), []*models.Action{
		act("conditions", map[string]any{"input": "$request.method$", "operator": "equals", "value": "DELETE", "action": "stop"}),
		act("set_variable", map[string]any{"name": "ran", "value": "yes"}),
	}, ec)
	if !ec.Stopped || len(res) != 1 {
		t.Errorf("stop condition not applied: stopped=%v results=%d", ec.Stopped, len(res))
	}

	// skip → next action skipped, then chain continues
	ec2 := e.NewContext(&models.Request{Method: "GET"}, testToken())
	e.Execute(context.Background(), []*models.Action{
		act("conditions", map[string]any{"input": "$request.method$", "operator": "equals", "value": "GET", "action": "skip"}),
		act("set_variable", map[string]any{"name": "skipped", "value": "yes"}),
		act("set_variable", map[string]any{"name": "after", "value": "yes"}),
	}, ec2)
	if _, ok := ec2.Vars["skipped"]; ok {
		t.Error("skipped action ran")
	}
	if ec2.Vars["after"] != "yes" {
		t.Error("chain did not continue after skip")
	}
}

func TestHTTPRequestForward(t *testing.T) {
	var gotMethod, gotBody, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b := make([]byte, r.ContentLength)
		r.Body.Read(b)
		gotBody = string(b)
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(202)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	// httptest binds loopback, so internal targets must be permitted here.
	e := New(WithSSRFLists(nil, nil, true))
	ec := e.NewContext(&models.Request{
		Method:  "POST",
		Content: "payload",
		Headers: map[string][]string{"Authorization": {"Bearer secret"}, "X-Event": {"push"}},
	}, testToken())
	res := e.Execute(context.Background(), []*models.Action{
		act("http_request", map[string]any{"url": srv.URL, "mode": "forward"}),
	}, ec)
	if res[0].Err != nil {
		t.Fatalf("http_request error: %v", res[0].Err)
	}
	if gotMethod != "POST" || gotBody != "payload" {
		t.Errorf("forwarded method/body = %q/%q", gotMethod, gotBody)
	}
	if gotAuth != "" {
		t.Errorf("Authorization header leaked to forward target: %q", gotAuth)
	}
	if ec.Vars["response.status"] != "202" {
		t.Errorf("response.status = %q, want 202", ec.Vars["response.status"])
	}
}

func TestHTTPRequestSSRFDenyList(t *testing.T) {
	e := New(WithSSRFLists(nil, []string{"169.254.169.254", "metadata.internal"}, true))
	ec := e.NewContext(&models.Request{}, testToken())
	res := e.Execute(context.Background(), []*models.Action{
		act("http_request", map[string]any{"url": "http://169.254.169.254/latest/meta-data/"}),
	}, ec)
	if res[0].Err == nil {
		t.Error("expected SSRF deny-list block")
	}
}

func TestHTTPRequestInternalBlockedByDefault(t *testing.T) {
	// Default guard blocks internal/loopback targets even with no deny list.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	e := New() // secure default: allowInternal = false
	ec := e.NewContext(&models.Request{}, testToken())
	res := e.Execute(context.Background(), []*models.Action{
		act("http_request", map[string]any{"url": srv.URL, "method": "GET"}),
	}, ec)
	if res[0].Err == nil {
		t.Error("expected internal target to be blocked by default")
	}
}
