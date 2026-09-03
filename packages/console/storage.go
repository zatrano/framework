package console

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zatrano/framework/kernel"
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
	dir := c.app.BasePath("tests")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+"_test.go")
	content := fmt.Sprintf(`package tests

import (
	"testing"

	"github.com/zatrano/framework/bootstrap"
	testkit "github.com/zatrano/packages/testing"
)

func Test%s(t *testing.T) {
	app := bootstrap.App()
	tc, err := testkit.New(app)
	if err != nil {
		t.Fatal(err)
	}

	tc.AcceptJSON().Get("/api/health").
		AssertOK().
		AssertJSONContains("status", "ok")
}
`, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Test created: %s\n", path)
	return nil
}
