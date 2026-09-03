package hashid

import "github.com/zatrano/framework/kernel"

// From resolves the package service from the application container.
func From(app *kernel.Application) *Hasher {
	return kernel.Resolve[*Hasher](app, "hashid")
}
