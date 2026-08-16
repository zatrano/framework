package bootstrap

// Optional SQL drivers — default is SQLite only (lean module).
// Add MySQL / PostgreSQL / SQL Server / Oracle with:
//
//	go run ./cmd/zatrano db:setup
//	go run ./cmd/zatrano db:setup --drivers=sqlite,mysql,pgsql --default=mysql --yes
import (
	_ "github.com/zatrano/framework/packages/database/driver/sqlite"
)
