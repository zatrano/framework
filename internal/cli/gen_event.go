package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/internal/gen"
)

var genEventCmd = &cobra.Command{
	Use:   "event [name]",
	Short: "Generate an event struct",
	Long: `Creates an event struct implementing events.Event:

  modules/events/<name>_event.go

Fire the generated event with:

  app.Events.Fire(ctx, &myevents.<Name>Event{})`,
	Args: cobra.ExactArgs(1),
	RunE: runGenEvent,
}

func init() {
	genEventCmd.Flags().String("out", "modules", "base directory for generated files")
	genEventCmd.Flags().String("module-root", ".", "directory containing go.mod")
	genEventCmd.Flags().Bool("dry-run", false, "print paths only, do not write files")
	genCmd.AddCommand(genEventCmd)
}

func runGenEvent(cmd *cobra.Command, args []string) error {
	out, _ := cmd.Flags().GetString("out")
	moduleRoot, _ := cmd.Flags().GetString("module-root")
	dry, _ := cmd.Flags().GetBool("dry-run")

	paths, err := gen.Event(moduleRoot, out, args[0], dry)
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
