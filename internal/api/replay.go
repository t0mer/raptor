package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/t0mer/raptor/internal/models"
)

// replayClient bounds replay delivery attempts.
var replayClient = &http.Client{Timeout: 15 * time.Second}

type replayRequest struct {
	TargetURL string `json:"target_url"`
	Query     string `json:"q"`
	DateFrom  string `json:"date_from"`
	DateTo    string `json:"date_to"`
	Limit     int    `json:"limit"`
}

// replayRequests re-delivers a subset of a token's captured requests to a target
// URL, preserving method, headers and body. Only the count is returned; target
// responses are not exposed.
func (a *API) replayRequests(w http.ResponseWriter, r *http.Request) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return
	}
	var body replayRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if strings.TrimSpace(body.TargetURL) == "" {
		writeError(w, http.StatusBadRequest, "target_url is required")
		return
	}

	filter := filterFrom(body.Query, body.DateFrom, body.DateTo)
	limit := body.Limit
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}

	reqs, err := a.store.ListRequestsWhere(r.Context(), tok.UUID, filter.SQL, filter.Args, 100, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load requests")
		return
	}

	replayed, failed := 0, 0
	for i, req := range reqs {
		if i >= limit {
			break
		}
		if err := replayOne(r.Context(), body.TargetURL, req); err != nil {
			failed++
			continue
		}
		replayed++
	}
	writeJSON(w, http.StatusOK, map[string]any{"replayed": replayed, "failed": failed})
}

func replayOne(ctx context.Context, target string, src *models.Request) error {
	method := src.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, target, strings.NewReader(src.Content))
	if err != nil {
		return err
	}
	for k, vals := range src.Headers {
		if skipReplayHeader(k) {
			continue
		}
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := replayClient.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func skipReplayHeader(k string) bool {
	switch strings.ToLower(k) {
	case "host", "content-length", "connection", "transfer-encoding", "keep-alive":
		return true
	}
	return false
}
