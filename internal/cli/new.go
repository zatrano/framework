package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/internal/cli/scaffold"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new ZATRANO application directory",
	Long: `Creates a runnable Go module that depends on github.com/zatrano/framework.

Use --replace when developing ZATRANO locally (points go.mod replace at your framework checkout).`,
	Args: cobra.ExactArgs(1),
	RunE: runNew,
}

func init() {
	newCmd.Flags().String("module", "", "Go module path (default: github.com/<name>/<name>)")
	newCmd.Flags().String("output", "", "output directory (default: ./<name>)")
	newCmd.Flags().String("replace-zatrano", "", "local path for `replace github.com/zatrano/framework => ...` in go.mod")
	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("name is required")
	}
	mod, _ := cmd.Flags().GetString("module")
	if mod == "" {
		mod = "github.com/" + name + "/" + name
	}
	out, _ := cmd.Flags().GetString("output")
	if out == "" {
		out = name
	}
	out, _ = filepath.Abs(out)
	rep, _ := cmd.Flags().GetString("replace-zatrano")
	if rep != "" {
		var err error
		rep, err = filepath.Abs(rep)
		if err != nil {
			return err
		}
	}

	if err := scaffold.Run(scaffold.Options{
		Dir:         out,
		AppName:     name,
		Module:      mod,
		ZatranoPath: rep,
	}); err != nil {
		return err
	}

	fmt.Printf("Created ZATRANO app in %s\n\n", out)
	fmt.Println("Next:")
	fmt.Println("  cd", filepath.Base(out))
	fmt.Println("  cp config/examples/dev.yaml config/dev.yaml")
	if rep == "" {
		fmt.Println("  go get github.com/zatrano/framework@latest   # or add replace-zatrano when hacking locally")
	}
	fmt.Println("  go mod tidy")
	fmt.Println("  go run ./cmd/" + name)
	return nil
}
