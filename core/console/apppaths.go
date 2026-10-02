package console

import (
	"github.com/zatrano/framework/v3/core/console/consolecore"
	"github.com/zatrano/framework/v3/core/kernel"
)

func consumerModule(app *kernel.Application) string {
	return consolecore.ConsumerModule(app)
}
