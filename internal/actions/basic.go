package actions

import (
	"context"
	"strconv"

	"github.com/t0mer/raptor/internal/models"
)

func init() {
	register("set_variable", newSetVariable)
	register("modify_response", newModifyResponse)
	register("stop", newStop)
	register("dont_save", newDontSave)
}

// set_variable — assign a (interpolated) value to a named variable.
type setVariable struct{ name, value string }

func newSetVariable(a *models.Action) (runner, error) {
	return &setVariable{name: strParam(a, "name"), value: strParam(a, "value")}, nil
}

func (s *setVariable) run(_ context.Context, ec *ExecContext) error {
	v := ec.Interp(s.value)
	ec.SetVar(s.name, v)
	ec.Logf("set %s = %s", s.name, v)
	return nil
}

// modify_response — override response status, content, content-type and headers.
type modifyResponse struct {
	status      string
	content     string
	contentType string
	headers     map[string]any
}

func newModifyResponse(a *models.Action) (runner, error) {
	hdr, _ := a.Parameters["headers"].(map[string]any)
	return &modifyResponse{
		status:      strParam(a, "status"),
		content:     strParam(a, "content"),
		contentType: strParam(a, "content_type"),
		headers:     hdr,
	}, nil
}

func (m *modifyResponse) run(_ context.Context, ec *ExecContext) error {
	if m.status != "" {
		if code, err := strconv.Atoi(ec.Interp(m.status)); err == nil {
			ec.Response.Status = code
		}
	}
	if m.content != "" {
		ec.Response.Content = ec.Interp(m.content)
	}
	if m.contentType != "" {
		ec.Response.ContentType = ec.Interp(m.contentType)
	}
	for k, v := range m.headers {
		if vs, ok := v.(string); ok {
			ec.Response.Headers[k] = ec.Interp(vs)
		}
	}
	ec.Response.Set = true
	ec.Logf("response set to %d", ec.Response.Status)
	return nil
}

// stop — halt the action chain.
type stop struct{}

func newStop(_ *models.Action) (runner, error) { return &stop{}, nil }

func (s *stop) run(_ context.Context, ec *ExecContext) error {
	ec.Stopped = true
	ec.Logf("chain stopped")
	return nil
}

// dont_save — record the request's actions but do not persist the request.
type dontSave struct{}

func newDontSave(_ *models.Action) (runner, error) { return &dontSave{}, nil }

func (d *dontSave) run(_ context.Context, ec *ExecContext) error {
	ec.DontSave = true
	ec.Logf("request will not be saved")
	return nil
}
