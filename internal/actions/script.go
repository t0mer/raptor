package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"

	"github.com/t0mer/raptor/internal/models"
)

func init() { register("script", newScript) }

// scriptTimeout bounds a single script's execution.
const scriptTimeout = 5 * time.Second

// script — run a JavaScript snippet (goja) with flow built-ins. Exposed API:
//
//	request            — {content, method, ip, query, headers}
//	get(name)/set(n,v) — read/write chain variables
//	respond(content[, status[, contentType]]) — set the response
//	stop()             — halt the chain
//	dont_save()        — do not persist the request
//	echo(...)          — write to the action output log
//	JSON.parse/stringify — native
type script struct{ src string }

func newScript(a *models.Action) (runner, error) {
	src := strParam(a, "script")
	if src == "" {
		src = strParam(a, "code")
	}
	if src == "" {
		return nil, fmt.Errorf("script: script is required")
	}
	return &script{src: src}, nil
}

func (s *script) run(ctx context.Context, ec *ExecContext) error {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	bindScriptAPI(vm, ec)

	// Enforce a wall-clock timeout via interrupt, honouring ctx cancellation too.
	timer := time.AfterFunc(scriptTimeout, func() { vm.Interrupt("script timeout") })
	defer timer.Stop()
	stop := context.AfterFunc(ctx, func() { vm.Interrupt("cancelled") })
	defer stop()

	if _, err := vm.RunString(s.src); err != nil {
		return fmt.Errorf("script error: %w", err)
	}
	return nil
}

func bindScriptAPI(vm *goja.Runtime, ec *ExecContext) {
	_ = vm.Set("request", map[string]any{
		"content": ec.Request.Content,
		"method":  ec.Request.Method,
		"ip":      ec.Request.IP,
		"query":   firstValues(ec.Request.Query),
		"headers": firstValues(ec.Request.Headers),
	})
	_ = vm.Set("get", func(name string) string { return ec.GetVar(name) })
	_ = vm.Set("set", func(name, value string) { ec.SetVar(name, value) })
	_ = vm.Set("stop", func() { ec.Stopped = true })
	_ = vm.Set("dont_save", func() { ec.DontSave = true })
	_ = vm.Set("echo", func(call goja.FunctionCall) goja.Value {
		parts := make([]any, len(call.Arguments))
		for i, a := range call.Arguments {
			parts[i] = a.String()
		}
		ec.Logf("%s", fmt.Sprint(parts...))
		return goja.Undefined()
	})
	_ = vm.Set("respond", func(call goja.FunctionCall) goja.Value {
		ec.Response.Content = call.Argument(0).String()
		if a := call.Argument(1); !goja.IsUndefined(a) {
			ec.Response.Status = int(a.ToInteger())
		}
		if a := call.Argument(2); !goja.IsUndefined(a) {
			ec.Response.ContentType = a.String()
		}
		ec.Response.Set = true
		return goja.Undefined()
	})
}

// firstValues flattens a multi-value map to its first values for JS ergonomics.
func firstValues(m map[string][]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}
