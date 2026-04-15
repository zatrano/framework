package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/internal/gen"
)

var genRequestCmd = &cobra.Command{
	Use:   "request [name]",
	Short: "Generate form request struct stubs under modules/<name>/requests/",
	Long: `Creates Create<Name>Request and Update<Name>Request structs with validation tags:

  modules/<name>/requests/create_<name>.go
  modules/<name>/requests/update_<name>.go

These structs are intended for use with zatrano.Validate[T](c) in handlers.`,
	Args: cobra.ExactArgs(1),
	RunE: runGenRequest,
}

func init() {
	genRequestCmd.Flags().String("out", "modules", "base directory for generated modules (relative to module-root)")
	genRequestCmd.Flags().String("module-root", ".", "directory containing go.mod")
	genRequestCmd.Flags().Bool("dry-run", false, "print paths only, do not write files")
	genCmd.AddCommand(genRequestCmd)
}

func runGenRequest(cmd *cobra.Command, args []string) error {
	out, _ := cmd.Flags().GetString("out")
	moduleRoot, _ := cmd.Flags().GetString("module-root")
	dry, _ := cmd.Flags().GetBool("dry-run")

	paths, err := gen.Request(moduleRoot, out, args[0], dry)
	if err != nil {
		return err
	}
	if dry {
		fmt.Println("dry-run — would write:")
	} else {
		fmt.Println("written:")
	}
	fmt.Println(strings.Join(paths, "\n"))
	return nil
}
