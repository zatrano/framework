package console

import "embed"

// Starter scaffolds embedded in the CLI so `go install` can run `zatrano new` offline.
//
//	templates/web   — zatrano new (HTML / plus JSON /api after API overlay)
//	templates/empty — overlay stub baseline for add:*
//	templates/api   — add:api overlay source
//	templates/overlays — additive files for add:api
//
// The generator engine lives in console/generator. These trees are content only.
// `zatrano new` is web + API overlay.
//
//go:embed all:templates
var starterTemplates embed.FS
