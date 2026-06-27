package actions

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/t0mer/raptor/internal/models"
)

func init() { register("http_request", newHTTPRequest) }

// maxResponseBytes caps how much of an outbound response is read into variables.
const maxResponseBytes = 1 << 20 // 1 MiB

// http_request — call out to another URL. In "forward" mode it proxies the
// captured request's method and body; otherwise it sends the configured body.
// The response status and body are stored in $<prefix>.status$ / $<prefix>.body$.
type httpRequest struct {
	url         string
	method      string
	mode        string
	body        string
	contentType string
	headers     map[string]any
	prefix      string
}

func newHTTPRequest(a *models.Action) (runner, error) {
	url := strParam(a, "url")
	if url == "" {
		return nil, fmt.Errorf("http_request: url is required")
	}
	hdr, _ := a.Parameters["headers"].(map[string]any)
	prefix := strParam(a, "response_var")
	if prefix == "" {
		prefix = "response"
	}
	return &httpRequest{
		url:         url,
		method:      strings.ToUpper(strParam(a, "method")),
		mode:        strings.ToLower(strParam(a, "mode")),
		body:        strParam(a, "body"),
		contentType: strParam(a, "content_type"),
		headers:     hdr,
		prefix:      prefix,
	}, nil
}

func (h *httpRequest) run(ctx context.Context, ec *ExecContext) error {
	url := ec.Interp(h.url)
	if err := ec.engineRef.ssrf.Check(url); err != nil {
		return fmt.Errorf("http_request blocked: %w", err)
	}

	method, body, contentType := h.resolve(ec)

	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range h.headers {
		if vs, ok := v.(string); ok {
			req.Header.Set(k, ec.Interp(vs))
		}
	}
	if h.mode == "forward" {
		copyForwardHeaders(req, ec.Request)
	}

	resp, err := ec.engineRef.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http_request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	ec.SetVar(h.prefix+".status", strconv.Itoa(resp.StatusCode))
	ec.SetVar(h.prefix+".body", string(respBody))
	ec.Logf("%s %s -> %d (%d bytes)", method, url, resp.StatusCode, len(respBody))
	return nil
}

func (h *httpRequest) resolve(ec *ExecContext) (method, body, contentType string) {
	if h.mode == "forward" {
		method = ec.Request.Method
		body = ec.Request.Content
		contentType = headerValue(ec.Request.Headers, "Content-Type")
		return
	}
	method = h.method
	if method == "" {
		method = http.MethodPost
	}
	body = ec.Interp(h.body)
	contentType = h.contentType
	if contentType == "" {
		contentType = "application/json"
	}
	return
}

// copyForwardHeaders copies the captured request's headers onto the outbound
// request, skipping hop-by-hop and host headers. Credential-bearing headers
// (Authorization, Cookie, Proxy-Authorization) are deliberately NOT forwarded,
// so the original caller's secrets are not leaked to the forward target.
func copyForwardHeaders(req *http.Request, src *models.Request) {
	skip := map[string]bool{
		"host": true, "content-length": true, "connection": true,
		"transfer-encoding": true, "keep-alive": true,
		"authorization": true, "cookie": true, "proxy-authorization": true,
	}
	for k, vals := range src.Headers {
		if skip[strings.ToLower(k)] {
			continue
		}
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
}
