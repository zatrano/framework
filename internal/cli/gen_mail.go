package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/internal/gen"
)

var genMailCmd = &cobra.Command{
	Use:   "mail [name]",
	Short: "Generate a Mailable struct and email template",
	Long: `Creates a Mailable struct and corresponding HTML template:

  modules/mails/<name>_mail.go    — Mailable struct with Build() method
  views/mails/<name>.html          — HTML email template

Send the generated mailable with:

  app.Mail.SendMailable(ctx, &mails.<Name>Mail{Name: "Alice", Email: "alice@example.com"})`,
	Args: cobra.ExactArgs(1),
	RunE: runGenMail,
}

func init() {
	genMailCmd.Flags().String("out", "modules", "base directory for generated mailable struct")
	genMailCmd.Flags().String("views", "views/mails", "directory for generated email templates")
	genMailCmd.Flags().String("module-root", ".", "directory containing go.mod")
	genMailCmd.Flags().Bool("dry-run", false, "print paths only, do not write files")
	genCmd.AddCommand(genMailCmd)
}

func runGenMail(cmd *cobra.Command, args []string) error {
	out, _ := cmd.Flags().GetString("out")
	views, _ := cmd.Flags().GetString("views")
	moduleRoot, _ := cmd.Flags().GetString("module-root")
	dry, _ := cmd.Flags().GetBool("dry-run")

	paths, err := gen.Mail(moduleRoot, out, views, args[0], dry)
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
		fmt.Printf("\nSend the mailable:\n")
		fmt.Printf("  app.Mail.SendMailable(ctx, &mails.%sMail{Name: \"Alice\", Email: \"alice@example.com\"})\n", pascal)
	}
	return nil
}
