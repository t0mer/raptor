package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/search"
	"github.com/t0mer/raptor/internal/store"
)

func (a *API) listRequests(w http.ResponseWriter, r *http.Request) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return
	}
	page := queryInt(r, "page", 1)
	if page < 1 {
		page = 1
	}
	perPage := queryInt(r, "per_page", 50)
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}

	filter := requestFilter(r)
	reqs, err := a.store.ListRequestsWhere(r.Context(), tok.UUID, filter.SQL, filter.Args, perPage, (page-1)*perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list requests")
		return
	}
	total, err := a.store.CountRequestsWhere(r.Context(), tok.UUID, filter.SQL, filter.Args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count requests")
		return
	}
	for _, req := range reqs {
		a.attachFiles(r, req)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     reqs,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (a *API) latestRequest(w http.ResponseWriter, r *http.Request) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return
	}
	req, err := a.store.LatestRequest(r.Context(), tok.UUID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no requests yet")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load request")
		return
	}
	a.attachFiles(r, req)
	writeJSON(w, http.StatusOK, req)
}

func (a *API) getRequest(w http.ResponseWriter, r *http.Request) {
	req, ok := a.loadRequest(w, r)
	if !ok {
		return
	}
	a.attachFiles(r, req)
	writeJSON(w, http.StatusOK, req)
}

func (a *API) rawRequest(w http.ResponseWriter, r *http.Request) {
	req, ok := a.loadRequest(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(rawHTTP(req)))
}

func (a *API) deleteRequest(w http.ResponseWriter, r *http.Request) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return
	}
	rid := chi.URLParam(r, "requestID")
	req, err := a.store.GetRequest(r.Context(), rid)
	if errors.Is(err, store.ErrNotFound) || (req != nil && req.TokenID != tok.UUID) {
		writeError(w, http.StatusNotFound, "request not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load request")
		return
	}
	if err := a.store.DeleteRequest(r.Context(), rid); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete request")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteAllRequests deletes a token's requests. With a search query (`q`) or
// `date_from`/`date_to` bounds it deletes only the matching subset; with none it
// deletes all.
func (a *API) deleteAllRequests(w http.ResponseWriter, r *http.Request) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return
	}
	filter := requestFilter(r)
	n, err := a.store.DeleteRequestsWhere(r.Context(), tok.UUID, filter.SQL, filter.Args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete requests")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": n})
}

func (a *API) downloadFile(w http.ResponseWriter, r *http.Request) {
	req, ok := a.loadRequest(w, r)
	if !ok {
		return
	}
	fileID := chi.URLParam(r, "fileID")
	f, err := a.store.GetFile(r.Context(), fileID)
	if errors.Is(err, store.ErrNotFound) || (f != nil && f.RequestID != req.UUID) {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load file")
		return
	}
	body, err := os.ReadFile(f.Path)
	if err != nil {
		writeError(w, http.StatusNotFound, "file blob missing")
		return
	}
	ct := f.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	// Force download and stop browsers from MIME-sniffing attacker-controlled
	// attachment content into an executable type.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", f.Filename))
	_, _ = w.Write(body)
}

// loadRequest resolves {requestID} and verifies it belongs to {tokenID}.
func (a *API) loadRequest(w http.ResponseWriter, r *http.Request) (*models.Request, bool) {
	tok, ok := a.loadToken(w, r)
	if !ok {
		return nil, false
	}
	rid := chi.URLParam(r, "requestID")
	req, err := a.store.GetRequest(r.Context(), rid)
	if errors.Is(err, store.ErrNotFound) || (req != nil && req.TokenID != tok.UUID) {
		writeError(w, http.StatusNotFound, "request not found")
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load request")
		return nil, false
	}
	return req, true
}

func (a *API) attachFiles(r *http.Request, req *models.Request) {
	files, err := a.store.ListFilesByRequest(r.Context(), req.UUID)
	if err == nil {
		req.Files = files
	}
}

// requestFilter compiles the search DSL (`q`) and optional `date_from`/`date_to`
// query parameters into a single combined filter.
func requestFilter(r *http.Request) search.Filter {
	q := r.URL.Query()
	return filterFrom(q.Get("q"), q.Get("date_from"), q.Get("date_to"))
}

// filterFrom combines a search-DSL query with optional ISO date bounds.
func filterFrom(query, dateFrom, dateTo string) search.Filter {
	f := search.Parse(query, time.Now().UTC())
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if f.SQL != "" {
		clauses = append(clauses, f.SQL)
		args = append(args, f.Args...)
	}
	if t, ok := parseDateParam(dateFrom); ok {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, t)
	}
	if t, ok := parseDateParam(dateTo); ok {
		clauses = append(clauses, "created_at <= ?")
		args = append(args, t)
	}
	return search.Filter{SQL: strings.Join(clauses, " AND "), Args: args}
}

// parseDateParam parses a YYYY-MM-DD or RFC3339 date into a comparable
// RFC3339Nano string (matching how created_at is stored).
func parseDateParam(s string) (string, bool) {
	if s == "" {
		return "", false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format(time.RFC3339Nano), true
		}
	}
	return "", false
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// rawHTTP reconstructs an approximate raw HTTP request representation.
func rawHTTP(req *models.Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", req.Method, req.URL)

	keys := make([]string, 0, len(req.Headers))
	for k := range req.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range req.Headers[k] {
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
	}
	b.WriteString("\n")
	b.WriteString(req.Content)
	return b.String()
}
