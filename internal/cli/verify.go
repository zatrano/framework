package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/openapi"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Run local checks (vet, test, merged OpenAPI) — typical pre-PR / CI gate",
	Long: `Runs in the current directory (expects go.mod):

  1. go vet ./...
  2. go test ./... -count=1 (add --race to catch data races; slower)
  3. merged OpenAPI validation (same as zatrano openapi validate --merged)

Use flags to skip steps when debugging. Requires Go toolchain on PATH.`,
	RunE: runVerify,
}

func init() {
	verifyCmd.Flags().Bool("no-vet", false, "skip go vet")
	verifyCmd.Flags().Bool("no-test", false, "skip go test")
	verifyCmd.Flags().Bool("race", false, "pass -race to go test (slower; use in CI or before release)")
	verifyCmd.Flags().Bool("no-openapi", false, "skip merged OpenAPI validation")
	verifyCmd.Flags().String("openapi-base", "api/openapi.yaml", "base spec path for merged validation")
	verifyCmd.Flags().String("module-root", ".", "directory containing go.mod (working directory for go commands)")
	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, _ []string) error {
	root, _ := cmd.Flags().GetString("module-root")
	noVet, _ := cmd.Flags().GetBool("no-vet")
	noTest, _ := cmd.Flags().GetBool("no-test")
	race, _ := cmd.Flags().GetBool("race")
	noOpenAPI, _ := cmd.Flags().GetBool("no-openapi")
	base, _ := cmd.Flags().GetString("openapi-base")

	root = filepath.Clean(root)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return fmt.Errorf("no go.mod in %q — run from the module root or use --module-root", root)
	}

	run := func(name string, fn func() error) error {
		fmt.Printf("→ %s\n", name)
		if err := fn(); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		fmt.Printf("  ok\n")
		return nil
	}

	goCmd := func(args ...string) error {
		c := exec.Command("go", args...)
		c.Dir = root
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Env = os.Environ()
		return c.Run()
	}

	if !noVet {
		if err := run("go vet ./...", func() error { return goCmd("vet", "./...") }); err != nil {
			return err
		}
	}
	if !noTest {
		name := "go test ./... -count=1"
		if race {
			name = "go test ./... -count=1 -race"
		}
		if err := run(name, func() error {
			args := []string{"test", "./...", "-count=1"}
			if race {
				args = append(args, "-race")
			}
			return goCmd(args...)
		}); err != nil {
			return err
		}
	}
	if !noOpenAPI {
		if err := run("openapi validate --merged", func() error {
			p := base
			if !filepath.IsAbs(p) {
				p = filepath.Join(root, p)
			}
			_, _, err := openapi.LoadMergedDocument(context.Background(), filepath.Clean(p))
			return err
		}); err != nil {
			return err
		}
	}

	fmt.Println("All verify steps passed.")
	return nil
}
