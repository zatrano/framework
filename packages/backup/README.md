# backup

Database backup and restore for every configured driver. SQLite uses a file copy; other engines call native CLI tools on `PATH`.

## CLI

```bash
go run . db:backup
go run . db:backup --connection=mysql --label=nightly
go run . db:backup:list
go run . db:restore backup_20260819_120000.sql --connection=mysql
```

## Required tools

| Driver | Backup | Restore |
|--------|--------|---------|
| sqlite | (file copy) | (file copy) |
| mysql | `mysqldump` | `mysql` |
| pgsql | `pg_dump` | `pg_restore` |
| mssql | `sqlpackage` / `SqlPackage` | same (Import) |
| oracle | `exp` | `imp` |
| mongo | `mongodump` | `mongorestore` |

Missing tools return a clear error — there is no silent fallback.

## Package API

```go
mgr := backup.NewManager(backup.Config{
    Driver: "pgsql", Host: "127.0.0.1", Port: "5432",
    Database: "app", Username: "postgres", Password: "x",
    Dir: "storage/backups",
})
path, err := mgr.Create("label")
err = mgr.Restore(filepath.Base(path))
```
