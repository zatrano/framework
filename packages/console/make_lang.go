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

// MakeLangCommand scaffolds a new locale directory from built-in defaults.
type MakeLangCommand struct {
	app *kernel.Application
}

func (c *MakeLangCommand) Name() string { return "make:lang" }
func (c *MakeLangCommand) Description() string {
	return "Create a new locale catalog under app/localization"
}
func (c *MakeLangCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("locale code required (example: make:lang de)")
	}
	locale := strings.ToLower(strings.TrimSpace(args[0]))
	group := ""
	rest := args[1:]
	for _, a := range rest {
		a = strings.TrimSpace(a)
		if strings.HasPrefix(a, "--group=") {
			group = strings.TrimSpace(strings.TrimPrefix(a, "--group="))
		}
	}
	if locale == "" || strings.ContainsAny(locale, `/\`) {
		return fmt.Errorf("invalid locale code")
	}
	if group != "" {
		if strings.ContainsAny(group, `/\`) {
			return fmt.Errorf("invalid group name")
		}
		dir := joinRoot(localizationRoot(c.app), locale)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		dest := filepath.Join(dir, group+".json")
		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("Skipped (exists): %s\n", dest)
			return nil
		}
		body := "{\n  \"example\": \"Example string\"\n}\n"
		if err := os.WriteFile(dest, []byte(body), 0o644); err != nil {
			return err
		}
		fmt.Printf("Created: %s\n", dest)
		return nil
	}

	source := "en"
	if _, err := defaults.FS.ReadDir(locale); err == nil {
		source = locale
	}

	created := 0
	err := fs.WalkDir(defaults.FS, source, func(path string, d fs.DirEntry, err error) error {
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
		rel := strings.TrimPrefix(filepath.ToSlash(path), source+"/")
		dest := joinRoot(localizationRoot(c.app), locale, filepath.FromSlash(rel))
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
		fmt.Printf("Created: %s\n", dest)
		created++
		return nil
	})
	if err != nil {
		// Fallback: copy flat files if en/ directory layout missing
		entries, readErr := defaults.FS.ReadDir(".")
		if readErr != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			raw, readErr := defaults.FS.ReadFile(entry.Name())
			if readErr != nil {
				continue
			}
			dest := joinRoot(localizationRoot(c.app), locale+".json")
			if writeErr := os.MkdirAll(filepath.Dir(dest), 0o755); writeErr != nil {
				return writeErr
			}
			if _, statErr := os.Stat(dest); statErr == nil {
				fmt.Printf("Skipped (exists): %s\n", dest)
				continue
			}
			if writeErr := os.WriteFile(dest, raw, 0o644); writeErr != nil {
				return writeErr
			}
			fmt.Printf("Created: %s\n", dest)
			created++
		}
	}
	if created == 0 {
		fmt.Printf("Locale %s already present or no templates found.\n", locale)
		return nil
	}
	fmt.Printf("Locale scaffold ready: lang/%s\n", locale)
	return nil
}
