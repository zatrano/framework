package console

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zatrano/framework/v3/core/console/generator"
	"github.com/zatrano/framework/v3/core/kernel"
)

func registerStorageCommands(console *Application, app *kernel.Application) {
	console.Register(
		&StorageLinkCommand{app: app},
		&MakeTestCommand{app: app},
	)
}

type StorageLinkCommand struct {
	app *kernel.Application
}

func (c *StorageLinkCommand) Name() string { return "storage:link" }
func (c *StorageLinkCommand) Description() string {
	return "Create the symbolic link for public storage"
}
func (c *StorageLinkCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	target := c.app.BasePath("storage", "app", "public")
	link := c.app.BasePath("public", "storage")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	if _, err := os.Lstat(link); err == nil {
		fmt.Println("The [public/storage] link already exists.")
		return nil
	}
	if err := os.Symlink(target, link); err != nil {
		// Fallback on Windows without symlink privileges: write a marker file.
		marker := []byte("Storage link target: " + target + "\n")
		if writeErr := os.WriteFile(link+".txt", marker, 0o644); writeErr != nil {
			return err
		}
		fmt.Println("Symlink unavailable; wrote public/storage.txt marker instead.")
		return nil
	}
	fmt.Println("The [public/storage] link has been connected.")
	return nil
}

type MakeTestCommand struct {
	app *kernel.Application
}

func (c *MakeTestCommand) Name() string        { return "make:test" }
func (c *MakeTestCommand) Description() string { return "Create a new test file" }
func (c *MakeTestCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("test name required")
	}
	name := args[0]
	path := filepath.Join(c.app.BasePath("tests"), toSnake(name)+"_test.go")
	content := fmt.Sprintf(`package tests

import (
	"testing"

	"github.com/zatrano/framework/v3/core/bootstrap"
	kernelhttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func Test%s(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_KEY", "zatrano-dev-key-do-not-use-prod!")
	app := bootstrap.App()
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	er, err := kernelhttp.ExchangeForTest(func(ctx *rawhttp.Ctx) {
		app.Handle(ctx)
	}, "GET /up HTTP/1.1\r\nHost: localhost\r\n\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if er.Status != 200 {
		t.Fatalf("GET /up: status %%d body=%%s", er.Status, er.Body)
	}
}
`, name)
	if err := generator.WriteFile(path, content); err != nil {
		return err
	}
	fmt.Printf("Test created: %s\n", path)
	return nil
}
