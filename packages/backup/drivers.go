package backup

import (
	"fmt"
	"os"

	"github.com/zatrano/framework/packages/database"
)

func (m *Manager) createMySQL(dest string) error {
	bin, err := lookPath("mysqldump")
	if err != nil {
		return err
	}
	host := m.cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := database.DefaultPort(m.cfg.Port, "3306")
	args := []string{
		"-h", host,
		"-P", port,
		"-u", m.cfg.Username,
		"--single-transaction",
		"--routines",
		"--triggers",
		"--result-file=" + dest,
		m.cfg.Database,
	}
	env := []string{}
	if m.cfg.Password != "" {
		env = append(env, "MYSQL_PWD="+m.cfg.Password)
	}
	return runCmd(bin, args, env, nil)
}

func (m *Manager) restoreMySQL(backupPath string) error {
	bin, err := lookPath("mysql")
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	host := m.cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := database.DefaultPort(m.cfg.Port, "3306")
	args := []string{
		"-h", host,
		"-P", port,
		"-u", m.cfg.Username,
		m.cfg.Database,
	}
	env := []string{}
	if m.cfg.Password != "" {
		env = append(env, "MYSQL_PWD="+m.cfg.Password)
	}
	if err := runCmd(bin, args, env, raw); err != nil {
		return err
	}
	return nil
}

func (m *Manager) createPostgres(dest string) error {
	bin, err := lookPath("pg_dump")
	if err != nil {
		return err
	}
	host := m.cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := database.DefaultPort(m.cfg.Port, "5432")
	ssl := m.cfg.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	args := []string{
		"-h", host,
		"-p", port,
		"-U", m.cfg.Username,
		"-d", m.cfg.Database,
		"-Fc",
		"-f", dest,
	}
	env := []string{"PGSSLMODE=" + ssl}
	if m.cfg.Password != "" {
		env = append(env, "PGPASSWORD="+m.cfg.Password)
	}
	return runCmd(bin, args, env, nil)
}

func (m *Manager) restorePostgres(backupPath string) error {
	bin, err := lookPath("pg_restore")
	if err != nil {
		return err
	}
	host := m.cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := database.DefaultPort(m.cfg.Port, "5432")
	ssl := m.cfg.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	args := []string{
		"-h", host,
		"-p", port,
		"-U", m.cfg.Username,
		"-d", m.cfg.Database,
		"--clean",
		"--if-exists",
		backupPath,
	}
	env := []string{"PGSSLMODE=" + ssl}
	if m.cfg.Password != "" {
		env = append(env, "PGPASSWORD="+m.cfg.Password)
	}
	return runCmd(bin, args, env, nil)
}

func (m *Manager) createMSSQL(dest string) error {
	bin, err := lookPath("sqlpackage", "SqlPackage")
	if err != nil {
		return fmt.Errorf("%w — install sqlpackage (SqlPackage) for MSSQL bacpac export", err)
	}
	cs := m.mssqlConnectionString()
	args := []string{
		"/Action:Export",
		"/SourceConnectionString:" + cs,
		"/TargetFile:" + dest,
	}
	return runCmd(bin, args, nil, nil)
}

func (m *Manager) restoreMSSQL(backupPath string) error {
	bin, err := lookPath("sqlpackage", "SqlPackage")
	if err != nil {
		return fmt.Errorf("%w — install sqlpackage (SqlPackage) for MSSQL bacpac import", err)
	}
	cs := m.mssqlConnectionString()
	args := []string{
		"/Action:Import",
		"/TargetConnectionString:" + cs,
		"/SourceFile:" + backupPath,
	}
	return runCmd(bin, args, nil, nil)
}

func (m *Manager) mssqlConnectionString() string {
	host := m.cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := database.DefaultPort(m.cfg.Port, "1433")
	return fmt.Sprintf(
		"Server=%s,%s;Database=%s;User Id=%s;Password=%s;TrustServerCertificate=True;",
		host, port, m.cfg.Database, m.cfg.Username, m.cfg.Password,
	)
}

func (m *Manager) createOracle(dest string) error {
	bin, err := lookPath("exp")
	if err != nil {
		return fmt.Errorf("%w — install Oracle Instant Client (exp) for Oracle export", err)
	}
	userid := m.oracleUserID()
	args := []string{
		userid,
		"file=" + dest,
		"owner=" + m.cfg.Username,
		"log=" + dest + ".log",
	}
	return runCmd(bin, args, nil, nil)
}

func (m *Manager) restoreOracle(backupPath string) error {
	bin, err := lookPath("imp")
	if err != nil {
		return fmt.Errorf("%w — install Oracle Instant Client (imp) for Oracle import", err)
	}
	userid := m.oracleUserID()
	args := []string{
		userid,
		"file=" + backupPath,
		"full=y",
		"ignore=y",
		"log=" + backupPath + ".import.log",
	}
	return runCmd(bin, args, nil, nil)
}

func (m *Manager) oracleUserID() string {
	service := m.cfg.Service
	if service == "" {
		service = m.cfg.Database
	}
	host := m.cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := database.DefaultPort(m.cfg.Port, "1521")
	// user/pass@//host:port/service
	return fmt.Sprintf("%s/%s@//%s:%s/%s", m.cfg.Username, m.cfg.Password, host, port, service)
}

func (m *Manager) createMongo(dest string) error {
	bin, err := lookPath("mongodump")
	if err != nil {
		return err
	}
	args := []string{"--archive=" + dest}
	args = append(args, m.mongoArgs()...)
	return runCmd(bin, args, nil, nil)
}

func (m *Manager) restoreMongo(backupPath string) error {
	bin, err := lookPath("mongorestore")
	if err != nil {
		return err
	}
	args := []string{"--archive=" + backupPath, "--drop"}
	args = append(args, m.mongoArgs()...)
	return runCmd(bin, args, nil, nil)
}

func (m *Manager) mongoArgs() []string {
	var args []string
	if m.cfg.URI != "" {
		args = append(args, "--uri="+m.cfg.URI)
	} else {
		host := m.cfg.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := database.DefaultPort(m.cfg.Port, "27017")
		args = append(args, "--host="+host+":"+port)
		if m.cfg.Username != "" {
			args = append(args, "--username="+m.cfg.Username)
		}
		if m.cfg.Password != "" {
			args = append(args, "--password="+m.cfg.Password)
		}
	}
	if m.cfg.Database != "" {
		args = append(args, "--db="+m.cfg.Database)
	}
	return args
}
