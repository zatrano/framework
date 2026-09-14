package console

import (
	"github.com/zatrano/framework/v2/console/consolecore"
	"github.com/zatrano/framework/v2/kernel"
)

func consumerModule(app *kernel.Application) string {
	return consolecore.ConsumerModule(app)
}
