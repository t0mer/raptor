package server

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	raptor "github.com/t0mer/raptor"
	"github.com/t0mer/raptor/internal/webui"
)

// placeholderHTML is served when no frontend build is embedded yet.
const placeholderHTML = `<!doctype html><html><head><meta charset="utf-8">
<title>Raptor</title></head><body style="font-family:sans-serif;max-width:40rem;margin:3rem auto">
<h1>Raptor</h1>
<p>The web UI has not been built into this binary yet.</p>
<ul>
<li>API: <code>/api/v1</code></li>
<li>API docs: <a href="/api/docs">/api/docs</a></li>
<li>Health: <a href="/health">/health</a></li>
</ul></body></html>`

// serveStaticAsset serves an embedded SPA file at the exact path if it exists.
// Returns true when it handled the request.
func serveStaticAsset(w http.ResponseWriter, r *http.Request, p string) bool {
	if !webui.Built() {
		return false
	}
	assets, err := webui.Assets()
	if err != nil {
		return false
	}
	clean := path.Clean(p)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return false
	}
	f, err := assets.Open(clean)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	http.ServeFileFS(w, r, assets, clean)
	return true
}

// serveSPAIndex serves the SPA shell (index.html) or the placeholder page.
func serveSPAIndex(w http.ResponseWriter, r *http.Request) {
	if webui.Built() {
		assets, err := webui.Assets()
		if err == nil {
			if f, err := assets.Open("index.html"); err == nil {
				defer f.Close()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = io.Copy(w, f)
				return
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, placeholderHTML)
}

// serveSpec serves the embedded OpenAPI document.
func serveSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(raptor.OpenAPISpec)
}

var _ fs.FS // keep io/fs import meaningful across build tags
