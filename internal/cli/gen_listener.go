package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/internal/gen"
)

var genListenerCmd = &cobra.Command{
	Use:   "listener [name]",
	Short: "Generate an event listener",
	Long: `Creates a listener struct implementing events.Listener:

  modules/listeners/<name>_listener.go

Use --queued to generate a ShouldQueue listener for async processing.

Register it with:

  app.Events.Listen("event_name", &listeners.<Name>Listener{})`,
	Args: cobra.ExactArgs(1),
	RunE: runGenListener,
}

func init() {
	genListenerCmd.Flags().String("out", "modules", "base directory for generated files")
	genListenerCmd.Flags().String("module-root", ".", "directory containing go.mod")
	genListenerCmd.Flags().Bool("queued", false, "generate a ShouldQueue listener for async processing")
	genListenerCmd.Flags().Bool("dry-run", false, "print paths only, do not write files")
	genCmd.AddCommand(genListenerCmd)
}

func runGenListener(cmd *cobra.Command, args []string) error {
	out, _ := cmd.Flags().GetString("out")
	moduleRoot, _ := cmd.Flags().GetString("module-root")
	queued, _ := cmd.Flags().GetBool("queued")
	dry, _ := cmd.Flags().GetBool("dry-run")

	paths, err := gen.Listener(moduleRoot, out, args[0], queued, dry)
	if err != nil {
		return err
	}
	if dry {
		fmt.Println("dry-run — would write:")
	} else {
		fmt.Println("written:")
	}
	fmt.Println(strings.Join(paths, "\n"))

	if !dry {
		name := gen.PackageName(args[0])
		pascal := snakeToPascalCLI(name)
		fmt.Printf("\nRegister the listener:\n")
		fmt.Printf("  app.Events.Listen(\"event_name\", &listeners.%sListener{})\n", pascal)
	}
	return nil
}
