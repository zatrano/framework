package backup

import (
	"fmt"
	"os"
	"strings"

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
	return runCmd(bin, args, env, nil, m.cfg.Password)
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
	return runCmd(bin, args, env, raw, m.cfg.Password)
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
	return runCmd(bin, args, env, nil, m.cfg.Password)
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
	return runCmd(bin, args, env, nil, m.cfg.Password)
}

func (m *Manager) createMSSQL(dest string) error {
	bin, err := lookPath("sqlpackage", "SqlPackage")
	if err != nil {
		return fmt.Errorf("%w — install sqlpackage (SqlPackage) for MSSQL bacpac export", err)
	}
	cs := m.mssqlConnectionString()
	body := "/Action:Export\n/SourceConnectionString:" + cs + "\n/TargetFile:" + dest + "\n"
	return withSecretFile("zatrano-sqlpackage-*.rsp", []byte(body), func(rsp string) error {
		return runCmd(bin, []string{"@" + rsp}, nil, nil, m.cfg.Password, cs)
	})
}

func (m *Manager) restoreMSSQL(backupPath string) error {
	bin, err := lookPath("sqlpackage", "SqlPackage")
	if err != nil {
		return fmt.Errorf("%w — install sqlpackage (SqlPackage) for MSSQL bacpac import", err)
	}
	cs := m.mssqlConnectionString()
	body := "/Action:Import\n/TargetConnectionString:" + cs + "\n/SourceFile:" + backupPath + "\n"
	return withSecretFile("zatrano-sqlpackage-*.rsp", []byte(body), func(rsp string) error {
		return runCmd(bin, []string{"@" + rsp}, nil, nil, m.cfg.Password, cs)
	})
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
	body := strings.Join([]string{
		"userid=" + userid,
		"file=" + dest,
		"owner=" + m.cfg.Username,
		"log=" + dest + ".log",
	}, "\n") + "\n"
	return withSecretFile("zatrano-oracle-*.par", []byte(body), func(par string) error {
		return runCmd(bin, []string{"PARFILE=" + par}, nil, nil, m.cfg.Password, userid)
	})
}

func (m *Manager) restoreOracle(backupPath string) error {
	bin, err := lookPath("imp")
	if err != nil {
		return fmt.Errorf("%w — install Oracle Instant Client (imp) for Oracle import", err)
	}
	userid := m.oracleUserID()
	body := strings.Join([]string{
		"userid=" + userid,
		"file=" + backupPath,
		"full=y",
		"ignore=y",
		"log=" + backupPath + ".import.log",
	}, "\n") + "\n"
	return withSecretFile("zatrano-oracle-*.par", []byte(body), func(par string) error {
		return runCmd(bin, []string{"PARFILE=" + par}, nil, nil, m.cfg.Password, userid)
	})
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
	return m.runMongo(bin, []string{"--archive=" + dest})
}

func (m *Manager) restoreMongo(backupPath string) error {
	bin, err := lookPath("mongorestore")
	if err != nil {
		return err
	}
	return m.runMongo(bin, []string{"--archive=" + backupPath, "--drop"})
}

func (m *Manager) runMongo(bin string, baseArgs []string) error {
	cfg, secrets := m.mongoConfigYAML()
	if strings.TrimSpace(cfg) == "" {
		return runCmd(bin, baseArgs, nil, nil)
	}
	return withSecretFile("zatrano-mongo-*.yaml", []byte(cfg), func(path string) error {
		args := append([]string{"--config=" + path}, baseArgs...)
		return runCmd(bin, args, nil, nil, secrets...)
	})
}

func (m *Manager) mongoConfigYAML() (string, []string) {
	var b strings.Builder
	var secrets []string
	if m.cfg.URI != "" {
		fmt.Fprintf(&b, "uri: %q\n", m.cfg.URI)
		secrets = append(secrets, m.cfg.URI)
		if m.cfg.Password != "" {
			secrets = append(secrets, m.cfg.Password)
		}
	} else {
		host := m.cfg.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := database.DefaultPort(m.cfg.Port, "27017")
		fmt.Fprintf(&b, "host: %q\n", host+":"+port)
		if m.cfg.Username != "" {
			fmt.Fprintf(&b, "username: %q\n", m.cfg.Username)
		}
		if m.cfg.Password != "" {
			fmt.Fprintf(&b, "password: %q\n", m.cfg.Password)
			secrets = append(secrets, m.cfg.Password)
		}
	}
	if m.cfg.Database != "" {
		fmt.Fprintf(&b, "db: %q\n", m.cfg.Database)
	}
	return b.String(), secrets
}
