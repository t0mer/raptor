// Command raptor is a self-hosted webhook/email/DNS capture and inspection
// server — a Go rewrite of webhook.site. See CLAUDE.md for the design contract.
package main

import (
	"fmt"
	"os"

	"github.com/t0mer/raptor/internal/version"
)

func main() {
	// Minimal bootstrap. Full flag/env configuration is wired up in
	// internal/config and consumed here in a subsequent change.
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v":
			fmt.Println(version.Version)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "raptor %s: server not yet wired up\n", version.Version)
}
