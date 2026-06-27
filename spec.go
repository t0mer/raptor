// Package raptor embeds repo-root assets that are the committed source of truth
// (currently the OpenAPI specification), so they ship inside the binary.
package raptor

import _ "embed"

// OpenAPISpec is the raw bytes of openapi.yaml, served at /api/openapi.yaml and
// rendered by the Swagger UI at /api/docs.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
