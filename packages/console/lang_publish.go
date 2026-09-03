package console

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/localization/defaults"
)

type LangPublishCommand struct {
	app *kernel.Application
}

func (c *LangPublishCommand) Name() string { return "lang:publish" }
func (c *LangPublishCommand) Description() string {
	return "Publish built-in language files into lang/"
}
func (c *LangPublishCommand) Handle(args []string) error {
	published := 0
	err := fs.WalkDir(defaults.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		raw, err := defaults.FS.ReadFile(path)
		if err != nil {
			return err
		}
		dest := c.app.BasePath("lang", filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("Skipped (exists): %s\n", dest)
			return nil
		}
		if err := os.WriteFile(dest, raw, 0o644); err != nil {
			return err
		}
		fmt.Printf("Published: %s\n", dest)
		published++
		return nil
	})
	if err != nil {
		return err
	}
	if published == 0 {
		fmt.Println("Language files already exist.")
	} else {
		fmt.Println("Locale switcher is available on the welcome page.")
	}
	return nil
}
