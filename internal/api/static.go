package api

import "embed"

// StaticFiles holds the built UI assets embedded at compile time.
// Populated by `make ui-build` (Vite outputs to internal/api/static/).
//
//go:embed static
var StaticFiles embed.FS
