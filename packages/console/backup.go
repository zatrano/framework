package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/backup"
)

func registerBackupCommands(console *Application, app *kernel.Application) {
	console.Register(
		&DBBackupCommand{app: app},
		&DBBackupListCommand{app: app},
		&DBRestoreCommand{app: app},
		&MakeProviderCommand{app: app},
	)
}

func backupManager(app *kernel.Application, args []string) (*backup.Manager, []string, error) {
	if err := app.Bootstrap(); err != nil {
		return nil, nil, err
	}
	connection, rest := parseBackupFlags(args)
	mgr, err := backup.ManagerFromApp(app, connection)
	if err != nil {
		return nil, nil, err
	}
	return mgr, rest, nil
}

func parseBackupFlags(args []string) (connection string, rest []string) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--connection" || arg == "-c":
			if i+1 < len(args) {
				connection = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--connection="):
			connection = strings.TrimPrefix(arg, "--connection=")
		case strings.HasPrefix(arg, "-c="):
			connection = strings.TrimPrefix(arg, "-c=")
		default:
			rest = append(rest, arg)
		}
	}
	return connection, rest
}

type DBBackupCommand struct{ app *kernel.Application }

func (c *DBBackupCommand) Name() string { return "db:backup" }
func (c *DBBackupCommand) Description() string {
	return "Backup the default (or --connection) database using native tools"
}
func (c *DBBackupCommand) Handle(args []string) error {
	mgr, rest, err := backupManager(c.app, args)
	if err != nil {
		return err
	}
	label := ""
	for i := 0; i < len(rest); i++ {
		if (rest[i] == "--label" || rest[i] == "-l") && i+1 < len(rest) {
			label = rest[i+1]
			i++
		}
	}
	path, err := mgr.Create(label)
	if err != nil {
		return err
	}
	fmt.Printf("Database backup created (%s): %s\n", mgr.Driver(), path)
	return nil
}

type DBBackupListCommand struct{ app *kernel.Application }

func (c *DBBackupListCommand) Name() string { return "db:backup:list" }
func (c *DBBackupListCommand) Description() string {
	return "List database backups for the default (or --connection) target directory"
}
func (c *DBBackupListCommand) Handle(args []string) error {
	mgr, _, err := backupManager(c.app, args)
	if err != nil {
		return err
	}
	files, err := mgr.List()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("No backups found.")
		return nil
	}
	for _, file := range files {
		info, _ := os.Stat(file)
		size := int64(0)
		mod := ""
		if info != nil {
			size = info.Size()
			mod = info.ModTime().Format(time.RFC3339)
		}
		fmt.Printf("%s\t%d bytes\t%s\n", filepath.Base(file), size, mod)
	}
	return nil
}

type DBRestoreCommand struct{ app *kernel.Application }

func (c *DBRestoreCommand) Name() string { return "db:restore" }
func (c *DBRestoreCommand) Description() string {
	return "Restore the default (or --connection) database from a backup file"
}
func (c *DBRestoreCommand) Handle(args []string) error {
	mgr, rest, err := backupManager(c.app, args)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return fmt.Errorf("backup filename required")
	}
	if err := mgr.Restore(rest[0]); err != nil {
		return err
	}
	fmt.Printf("Database restored (%s) from %s\n", mgr.Driver(), rest[0])
	return nil
}

type MakeProviderCommand struct{ app *kernel.Application }

func (c *MakeProviderCommand) Name() string        { return "make:provider" }
func (c *MakeProviderCommand) Description() string { return "Create a new service provider" }
func (c *MakeProviderCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provider name required")
	}
	name := args[0]
	if !strings.HasSuffix(name, "ServiceProvider") {
		name += "ServiceProvider"
	}
	dir := c.app.BasePath("app", "providers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("provider already exists: %s", path)
	}
	content := fmt.Sprintf(`package providers

import "github.com/zatrano/framework/kernel"

// %s registers application services.
type %s struct{}

func (p *%s) Register(app *kernel.Application) {
	// Bind services into the container.
}

func (p *%s) Boot(app *kernel.Application) {
	// Boot services after all providers are registered.
}
`, name, name, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Provider created: %s\n", path)
	fmt.Println("Remember to register it in bootstrap/app.go")
	return nil
}
