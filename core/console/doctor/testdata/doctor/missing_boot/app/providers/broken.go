package providers

import "github.com/zatrano/framework/v3/core/kernel"

type BrokenProvider struct{}

func (p *BrokenProvider) Register(app *kernel.Application) error {
	return nil
}
