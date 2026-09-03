package console

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/database"
)

func registerDBSetupCommand(console *Application, app *kernel.Application) {
	console.Register(&DBSetupCommand{app: app})
}

// DBSetupCommand interactively (or via flags) selects SQL drivers, writes
// bootstrap/database_drivers.go, updates .env, and runs go get for selected modules.
type DBSetupCommand struct {
	app *kernel.Application
}

func (c *DBSetupCommand) Name() string { return "db:setup" }
func (c *DBSetupCommand) Description() string {
	return "Choose databases (SQL + Mongo, single or multi), link drivers, write .env"
}
func (c *DBSetupCommand) Handle(args []string) error {
	driversFlag := flagValue(args, "--drivers")
	defaultFlag := flagValue(args, "--default")
	noInteractive := hasFlag(args, "--no-interactive", "-y", "--yes")

	var selected []string
	var defaultName string

	if driversFlag != "" {
		for _, p := range strings.Split(driversFlag, ",") {
			n := database.NormalizeDriverName(p)
			if n == "" {
				continue
			}
			if database.DriverModulePath(n) == "" {
				return fmt.Errorf("unknown driver %q (want: %s)", p, strings.Join(database.KnownDrivers(), ", "))
			}
			selected = append(selected, n)
		}
		selected = uniqueStrings(selected)
		defaultName = database.NormalizeDriverName(defaultFlag)
		if defaultName == "" && len(selected) > 0 {
			defaultName = selected[0]
		}
	} else if noInteractive {
		selected = []string{"sqlite"}
		defaultName = "sqlite"
	} else {
		var err error
		selected, defaultName, err = promptDatabaseSelection()
		if err != nil {
			return err
		}
	}

	if len(selected) == 0 {
		return fmt.Errorf("select at least one driver")
	}
	if defaultName == "" {
		defaultName = selected[0]
	}
	found := false
	for _, s := range selected {
		if s == defaultName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("default %q must be one of the selected drivers", defaultName)
	}

	if err := writeDatabaseDriversFile(c.app.BasePath("bootstrap", "database_drivers.go"), selected); err != nil {
		return err
	}
	fmt.Println("Wrote bootstrap/database_drivers.go")

	if err := goGetDrivers(c.app.BasePath(), selected); err != nil {
		return err
	}

	if err := updateEnvDatabase(c.app.BasePath(".env"), selected, defaultName); err != nil {
		fmt.Printf("Note: could not update .env (%v) — set DB_CONNECTION / DB_CONNECTIONS manually.\n", err)
	} else {
		fmt.Println("Updated .env (DB_CONNECTION / DB_CONNECTIONS)")
	}

	fmt.Println()
	fmt.Println("Enabled connections:", strings.Join(selected, ", "))
	fmt.Println("Default:", defaultName)
	fmt.Println("Multi-DB env: DB_CONNECTIONS=mysql,pgsql,mongo  DB_MYSQL_HOST=…  DB_PGSQL_DATABASE=…  DB_MONGO_URI=…")
	fmt.Println(`Model: func (m *Order) Connection() string { return "pgsql" }`)
	fmt.Println(`CLI:   go run ./cmd/zatrano make:model Order --connection=pgsql`)
	hasSQL := false
	for _, s := range selected {
		if !database.IsDocumentStore(s) {
			hasSQL = true
			break
		}
	}
	if hasSQL {
		fmt.Println("Next: go run ./cmd/zatrano db:create && go run ./cmd/zatrano migrate")
	} else {
		fmt.Println(`Next: resolve container key "mongo" (document API ready)`)
	}
	return nil
}

func promptDatabaseSelection() ([]string, string, error) {
	in := bufio.NewReader(os.Stdin)
	known := database.KnownDrivers()
	fmt.Println("ZATRANO database setup")
	fmt.Println("Select drivers (comma-separated numbers or names). Multi-DB is supported.")
	for i, name := range known {
		fmt.Printf("  %d) %s\n", i+1, name)
	}
	fmt.Print("Drivers [1=sqlite]: ")
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		line = "1"
	}
	selected := parseDriverSelection(line, known)
	if len(selected) == 0 {
		return nil, "", fmt.Errorf("no valid drivers selected")
	}

	fmt.Printf("Default connection %v: ", selected)
	defLine, _ := in.ReadString('\n')
	defLine = strings.TrimSpace(defLine)
	defaultName := database.NormalizeDriverName(defLine)
	if defaultName == "" {
		defaultName = selected[0]
	}
	ok := false
	for _, s := range selected {
		if s == defaultName {
			ok = true
			break
		}
	}
	if !ok {
		// allow index into selected
		if idx := atoiSafe(defLine); idx >= 1 && idx <= len(selected) {
			defaultName = selected[idx-1]
			ok = true
		}
	}
	if !ok {
		return nil, "", fmt.Errorf("default must be one of %v", selected)
	}
	return selected, defaultName, nil
}

func parseDriverSelection(line string, known []string) []string {
	parts := strings.Split(line, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var name string
		if idx := atoiSafe(p); idx >= 1 && idx <= len(known) {
			name = known[idx-1]
		} else {
			name = database.NormalizeDriverName(p)
		}
		if name == "" || database.DriverModulePath(name) == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func writeDatabaseDriversFile(path string, drivers []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("package bootstrap\n\n")
	b.WriteString("// Code generated by `zatrano db:setup`. DO NOT edit by hand unless you know why.\n")
	b.WriteString("// Re-run: go run ./cmd/zatrano db:setup --drivers=...\n")
	b.WriteString("import (\n")
	for _, d := range drivers {
		mod := database.DriverModulePath(d)
		b.WriteString("\t_ \"" + mod + "\"\n")
	}
	b.WriteString(")\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func goGetDrivers(basePath string, drivers []string) error {
	args := []string{"get"}
	for _, d := range drivers {
		mod := database.DriverModulePath(d)
		if mod == "" {
			continue
		}
		// Local replace path for framework checkout; published tags use module path@version.
		local := filepath.Join(basePath, "packages", "database", "driver", d)
		if st, err := os.Stat(local); err == nil && st.IsDir() {
			args = append(args, mod+"@v1.0.0")
			continue
		}
		args = append(args, mod+"@v1.0.0")
	}
	if len(args) == 1 {
		return nil
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = basePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Ensure replace directives for local driver modules.
		if err := ensureDriverReplaces(basePath, drivers); err != nil {
			return fmt.Errorf("go get drivers: %w", err)
		}
		cmd = exec.Command("go", "mod", "tidy")
		cmd.Dir = basePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
		return nil
	}
	_ = ensureDriverReplaces(basePath, drivers)
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = basePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ensureDriverReplaces(basePath string, drivers []string) error {
	modPath := filepath.Join(basePath, "go.mod")
	body, err := os.ReadFile(modPath)
	if err != nil {
		return err
	}
	src := string(body)
	changed := false
	for _, d := range drivers {
		mod := database.DriverModulePath(d)
		if strings.HasPrefix(mod, "github.com/zatrano/framework/") {
			rel := "./packages/database/driver/" + d
			line := fmt.Sprintf("\t%s => %s\n", mod, rel)
			if strings.Contains(src, mod+" =>") {
				continue
			}
			if strings.Contains(src, "replace (") {
				src = strings.Replace(src, "replace (", "replace (\n"+line, 1)
			} else if strings.Contains(src, "\nreplace ") {
				src = src + "\nreplace (\n" + line + ")\n"
			} else {
				src = src + "\nreplace (\n" + line + ")\n"
			}
		} else if strings.Contains(src, mod+" =>") {
			continue
		}
		// also require
		req := fmt.Sprintf("\t%s v1.0.0\n", mod)
		if !strings.Contains(src, mod+" ") {
			if idx := strings.Index(src, "require ("); idx >= 0 {
				src = src[:idx+9] + "\n" + req + src[idx+9:]
			}
		}
		changed = true
	}
	if !changed {
		return nil
	}
	return os.WriteFile(modPath, []byte(src), 0o644)
}

func updateEnvDatabase(path string, drivers []string, defaultName string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			content := fmt.Sprintf("DB_CONNECTION=%s\nDB_CONNECTIONS=%s\n", defaultName, strings.Join(drivers, ","))
			content += envDriverHints(drivers)
			return os.WriteFile(path, []byte(content), 0o644)
		}
		return err
	}
	lines := strings.Split(string(body), "\n")
	out := make([]string, 0, len(lines)+8)
	seenConn, seenList := false, false
	seen := map[string]bool{}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		key := envKeyOf(trim)
		if key != "" {
			seen[key] = true
		}
		if strings.HasPrefix(trim, "DB_CONNECTION=") {
			out = append(out, "DB_CONNECTION="+defaultName)
			seenConn = true
			continue
		}
		if strings.HasPrefix(trim, "DB_CONNECTIONS=") {
			out = append(out, "DB_CONNECTIONS="+strings.Join(drivers, ","))
			seenList = true
			continue
		}
		out = append(out, line)
	}
	if !seenConn {
		out = append(out, "DB_CONNECTION="+defaultName)
	}
	if !seenList {
		out = append(out, "DB_CONNECTIONS="+strings.Join(drivers, ","))
	}
	for _, line := range strings.Split(strings.TrimSuffix(envDriverHints(drivers), "\n"), "\n") {
		if line == "" {
			continue
		}
		key := envKeyOf(line)
		if key != "" && seen[key] {
			continue
		}
		out = append(out, line)
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

func envKeyOf(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	if i := strings.IndexByte(line, '='); i > 0 {
		return line[:i]
	}
	return ""
}

func envDriverHints(drivers []string) string {
	var b strings.Builder
	for _, d := range drivers {
		switch d {
		case "mongo":
			b.WriteString("DB_MONGO_URI=memory\n")
			b.WriteString("MONGO_URI=memory\n")
			b.WriteString("DB_MONGO_DATABASE=zatrano\n")
		case "mysql":
			b.WriteString("# DB_MYSQL_HOST=127.0.0.1\n")
			b.WriteString("# DB_MYSQL_PORT=3306\n")
			b.WriteString("# DB_MYSQL_DATABASE=zatrano\n")
			b.WriteString("# DB_MYSQL_USERNAME=root\n")
			b.WriteString("# DB_MYSQL_PASSWORD=\n")
		case "pgsql":
			b.WriteString("# DB_PGSQL_HOST=127.0.0.1\n")
			b.WriteString("# DB_PGSQL_PORT=5432\n")
			b.WriteString("# DB_PGSQL_DATABASE=zatrano\n")
			b.WriteString("# DB_PGSQL_USERNAME=postgres\n")
			b.WriteString("# DB_PGSQL_PASSWORD=\n")
			b.WriteString("# DB_PGSQL_SSLMODE=disable\n")
		case "mssql":
			b.WriteString("# DB_MSSQL_HOST=127.0.0.1\n")
			b.WriteString("# DB_MSSQL_PORT=1433\n")
			b.WriteString("# DB_MSSQL_DATABASE=zatrano\n")
			b.WriteString("# DB_MSSQL_USERNAME=sa\n")
			b.WriteString("# DB_MSSQL_PASSWORD=\n")
		case "oracle":
			b.WriteString("# DB_ORACLE_HOST=127.0.0.1\n")
			b.WriteString("# DB_ORACLE_PORT=1521\n")
			b.WriteString("# DB_ORACLE_SERVICE=FREEPDB1\n")
			b.WriteString("# DB_ORACLE_USERNAME=system\n")
			b.WriteString("# DB_ORACLE_PASSWORD=\n")
		}
	}
	return b.String()
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, name+"=") {
			return strings.TrimPrefix(a, name+"=")
		}
	}
	return ""
}
