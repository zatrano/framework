package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/internal/gen"
)

var genJobCmd = &cobra.Command{
	Use:   "job [name]",
	Short: "Generate a queue job stub under modules/jobs/",
	Long: `Creates a job file that implements queue.Job with Handle, Name, Queue,
Retries, and Timeout methods:

  modules/jobs/<name>.go

Register the generated job with the queue manager:

  queueManager.Register("<name>", func() queue.Job { return &jobs.<Name>Job{} })`,
	Args: cobra.ExactArgs(1),
	RunE: runGenJob,
}

func init() {
	genJobCmd.Flags().String("out", "modules", "base directory for generated files")
	genJobCmd.Flags().String("module-root", ".", "directory containing go.mod")
	genJobCmd.Flags().Bool("dry-run", false, "print paths only, do not write files")
	genCmd.AddCommand(genJobCmd)
}

func runGenJob(cmd *cobra.Command, args []string) error {
	out, _ := cmd.Flags().GetString("out")
	moduleRoot, _ := cmd.Flags().GetString("module-root")
	dry, _ := cmd.Flags().GetBool("dry-run")

	paths, err := gen.Job(moduleRoot, out, args[0], dry)
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
		fmt.Printf("\nRegister the job in your application startup:\n")
		fmt.Printf("  queueManager.Register(%q, func() queue.Job { return &jobs.%sJob{} })\n", name, pascal)
	}
	return nil
}
