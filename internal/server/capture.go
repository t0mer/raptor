package server

import (
	"net/http"
	"strconv"
	"strings"
)

// handleCaptureOrSPA is the root catch-all. Precedence:
//  1. an embedded SPA static asset at the exact path,
//  2. a token (UUID or alias) as the first path segment → capture,
//  3. otherwise serve the SPA shell for browser navigations (client-side
//     routing), or 404 for non-HTML requests.
func (s *Server) handleCaptureOrSPA(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if path == "" {
		serveSPAIndex(w, r)
		return
	}

	if serveStaticAsset(w, r, path) {
		return
	}

	segments := strings.SplitN(path, "/", 3)
	first := segments[0]

	tok, err := s.capturer.Resolve(r.Context(), first)
	if err == nil {
		var override *int
		// /{tokenId}/{statusCode} overrides the default response status.
		if len(segments) == 2 {
			if code, ok := parseStatusCode(segments[1]); ok {
				override = &code
			}
		}
		s.capturer.Handle(w, r, tok, override)
		return
	}

	// Unknown first segment: let the SPA handle the route for browser GETs.
	if isHTMLGet(r) {
		serveSPAIndex(w, r)
		return
	}
	http.NotFound(w, r)
}

func parseStatusCode(s string) (int, bool) {
	code, err := strconv.Atoi(s)
	if err != nil || code < 100 || code > 599 {
		return 0, false
	}
	return code, true
}

func isHTMLGet(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html") || r.Header.Get("Accept") == ""
}
