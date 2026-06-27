package actions

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/t0mer/raptor/internal/models"
)

// Engine executes action chains. It is safe for concurrent use.
type Engine struct {
	httpClient *http.Client
	ssrf       *ssrfGuard
}

// Option configures an Engine.
type Option func(*Engine)

// WithHTTPClient overrides the client used by outbound HTTP actions.
func WithHTTPClient(c *http.Client) Option {
	return func(e *Engine) {
		if c != nil {
			e.httpClient = c
		}
	}
}

// WithSSRFLists restricts outbound HTTP/script hosts. allow (if non-empty) is a
// strict allow-list; deny always blocks. Internal targets are blocked unless
// allowInternal is true. See ssrf.go.
func WithSSRFLists(allow, deny []string, allowInternal bool) Option {
	return func(e *Engine) { e.ssrf = newSSRFGuard(allow, deny, allowInternal) }
}

// New builds an Engine. Unless WithHTTPClient overrides it, the outbound HTTP
// client is constructed from the SSRF guard so the deny-list is enforced against
// resolved IPs and across redirects.
func New(opts ...Option) *Engine {
	e := &Engine{ssrf: newSSRFGuard(nil, nil, false)}
	for _, o := range opts {
		o(e)
	}
	if e.httpClient == nil {
		e.httpClient = e.ssrf.client(15 * time.Second)
	}
	return e
}

// RunResult is the outcome of one executed action.
type RunResult struct {
	Action  *models.Action
	Output  string
	Err     error
	Skipped bool
}

// NewContext seeds an ExecContext from a request and the token's default
// response. Actions mutate this context as the chain runs.
func (e *Engine) NewContext(req *models.Request, tok *models.Token) *ExecContext {
	resp := &Response{
		Status:      tok.DefaultStatus,
		Content:     tok.DefaultContent,
		ContentType: tok.DefaultContentType,
		Headers:     map[string]string{},
	}
	if resp.Status == 0 {
		resp.Status = http.StatusOK
	}
	if resp.ContentType == "" {
		resp.ContentType = "text/plain"
	}
	return &ExecContext{
		Request:  req,
		Vars:     map[string]string{},
		Response: resp,
	}
}

// Execute runs the actions in order against ec, returning a result per executed
// action. Disabled actions are skipped silently; a stop halts the chain.
func (e *Engine) Execute(ctx context.Context, acts []*models.Action, ec *ExecContext) []RunResult {
	var results []RunResult
	for _, a := range acts {
		if a.Disabled {
			continue
		}
		if ec.skipNext {
			ec.skipNext = false
			results = append(results, RunResult{Action: a, Skipped: true})
			continue
		}

		var buf strings.Builder
		ec.out = &buf
		ec.engineRef = e

		r := RunResult{Action: a}
		runnerFn, err := e.build(a)
		if err != nil {
			r.Err = err
		} else {
			r.Err = runnerFn.run(ctx, ec)
		}
		r.Output = buf.String()
		results = append(results, r)

		if ec.Stopped {
			break
		}
	}
	ec.out = nil
	return results
}

func (e *Engine) build(a *models.Action) (runner, error) {
	f, ok := registry[a.Type]
	if !ok {
		return nil, errUnknownType(a.Type)
	}
	return f(a)
}

type errUnknownType string

func (e errUnknownType) Error() string { return "unknown action type: " + string(e) }
