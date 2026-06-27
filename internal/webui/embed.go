// Package webui embeds the built React/Vite single-page application.
//
// The Vite build writes its output straight into the dist/ directory, which is
// embedded into the binary at compile time. A tracked dist/.gitkeep placeholder
// ensures this package compiles before any frontend build has run; the "all:"
// embed prefix is required so that dotfile is included.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Assets returns the embedded SPA file system rooted at the dist directory.
func Assets() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// Built reports whether a real frontend build is embedded (i.e. an index.html
// exists). When false, only the .gitkeep placeholder is present and the server
// should serve an API-only fallback.
func Built() bool {
	sub, err := Assets()
	if err != nil {
		return false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return false
	}
	return true
}
