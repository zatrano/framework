package console

import "embed"

// Starter scaffolds embedded in the CLI so `go install` can run `zatrano new` offline.
//
//	templates/empty — default unopinionated application (`zatrano new`)
//	templates/web   — web presentation (`zatrano new --web`)
//	templates/api   — API presentation (`zatrano new --api`)
//	templates/overlays — additive files for add:web / add:api / --full
//
// The generator engine lives in console/generator. These trees are content only.
// `new --full` is web + API overlay, not a fourth copied tree.
//
//go:embed all:templates
var starterTemplates embed.FS
