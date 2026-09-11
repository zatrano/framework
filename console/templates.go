package console

import "embed"

// Starter templates embedded in the CLI so `go install` can run `zatrano new` offline.
//
//	templates/web — HTML at / and JSON at /api
//
// The generator engine lives in console/generator. This tree is content only.
//
//go:embed all:templates
var starterTemplates embed.FS
