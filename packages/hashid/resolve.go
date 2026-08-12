package hashid

import "github.com/zatrano/framework/core"

// From resolves the package service from the application container.
func From(app *core.Application) *Hasher {
	return core.Resolve[*Hasher](app, "hashid")
}
