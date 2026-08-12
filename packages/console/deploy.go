package console

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/core"
)

func registerDeployCommands(console *Application, app *core.Application) {
	console.Register(
		&DeployCheckCommand{app: app},
		&DeployBuildCommand{app: app},
	)
}

type DeployCheckCommand struct {
	app *core.Application
}

func (c *DeployCheckCommand) Name() string        { return "deploy:check" }
func (c *DeployCheckCommand) Description() string { return "Validate deployment readiness" }
func (c *DeployCheckCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}

	issues := make([]string, 0)
	key := c.app.Config().GetString("app.key")
	if key == "" {
		issues = append(issues, "APP_KEY is empty (run key:generate)")
	}
	if c.app.Config().GetString("app.env", "local") == "production" && c.app.IsDebug() {
		issues = append(issues, "APP_DEBUG should be false in production")
	}
	for _, dir := range []string{
		c.app.BasePath("storage", "logs"),
		c.app.BasePath("storage", "framework", "sessions"),
		c.app.BasePath("storage", "framework", "cache"),
		c.app.BasePath("public"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			issues = append(issues, "cannot create "+dir+": "+err.Error())
		}
	}
	if _, err := os.Stat(c.app.BasePath("Dockerfile")); err != nil {
		issues = append(issues, "Dockerfile missing")
	}
	if _, err := os.Stat(c.app.BasePath("docker-compose.yml")); err != nil {
		issues = append(issues, "docker-compose.yml missing")
	}

	if len(issues) == 0 {
		fmt.Println("deploy:check passed")
		return nil
	}
	fmt.Println("deploy:check found issues:")
	for _, issue := range issues {
		fmt.Println(" -", issue)
	}
	return fmt.Errorf("%d deployment issue(s)", len(issues))
}

type DeployBuildCommand struct {
	app *core.Application
}

func (c *DeployBuildCommand) Name() string { return "deploy:build" }
func (c *DeployBuildCommand) Description() string {
	return "Build the application binary or Docker image"
}
func (c *DeployBuildCommand) Handle(args []string) error {
	mode := "binary"
	tag := "zatrano:latest"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--docker":
			mode = "docker"
		case "--tag":
			if i+1 < len(args) {
				tag = args[i+1]
				i++
			}
		}
	}

	if mode == "docker" {
		cmd := exec.Command("docker", "build", "-t", tag, ".")
		cmd.Dir = c.app.BasePath()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	out := "zatrano"
	if filepath.Ext(out) == "" && strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		out = "zatrano.exe"
	}
	cmd := exec.Command("go", "build", "-o", out, "./cmd/zatrano")
	cmd.Dir = c.app.BasePath()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Println("binary built:", filepath.Join(c.app.BasePath(), out))
	return nil
}
