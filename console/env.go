package console

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/v2/kernel"
)

func registerEnvCommands(console *Application, app *kernel.Application) {
	console.Register(
		&EnvEncryptCommand{app: app},
		&EnvDecryptCommand{app: app},
	)
}

type EnvEncryptCommand struct {
	app *kernel.Application
}

func (c *EnvEncryptCommand) Name() string        { return "env:encrypt" }
func (c *EnvEncryptCommand) Description() string { return "Encrypt a value using APP_KEY" }
func (c *EnvEncryptCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: env:encrypt <value>")
	}
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	value := strings.Join(args, " ")
	out, err := c.app.Encrypter().Encrypt(value)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

type EnvDecryptCommand struct {
	app *kernel.Application
}

func (c *EnvDecryptCommand) Name() string        { return "env:decrypt" }
func (c *EnvDecryptCommand) Description() string { return "Decrypt a value using APP_KEY" }
func (c *EnvDecryptCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: env:decrypt <payload>")
	}
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	out, err := c.app.Encrypter().Decrypt(args[0])
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
