// Package version exposes the build-time version string for Raptor.
//
// Version is injected at build time via:
//
//	-ldflags "-X github.com/t0mer/raptor/internal/version.Version=<v>"
package version

// Version is the application version, set at build time. It defaults to "dev"
// for local (non-release) builds.
var Version = "dev"
