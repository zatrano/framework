package defaults

import "embed"

// FS holds built-in locale JSON packs (messages + validation).
//
//go:embed en/*.json tr/*.json
var FS embed.FS
