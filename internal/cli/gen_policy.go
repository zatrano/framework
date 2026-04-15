package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/internal/gen"
)

var genPolicyCmd = &cobra.Command{
	Use:   "policy [name]",
	Short: "Generate an authorization policy stub under modules/<name>/policies/",
	Long: `Creates a policy file that implements auth.Policy with CRUD authorization methods:

  modules/<name>/policies/<name>_policy.go

The generated policy contains ViewAny, View, Create, Update, Delete, ForceDelete,
and Restore methods. Register the policy with:

  gate.RegisterPolicy("<name>", &policies.<Name>Policy{})`,
	Args: cobra.ExactArgs(1),
	RunE: runGenPolicy,
}

func init() {
	genPolicyCmd.Flags().String("out", "modules", "base directory for generated modules (relative to module-root)")
	genPolicyCmd.Flags().String("module-root", ".", "directory containing go.mod")
	genPolicyCmd.Flags().Bool("dry-run", false, "print paths only, do not write files")
	genCmd.AddCommand(genPolicyCmd)
}

func runGenPolicy(cmd *cobra.Command, args []string) error {
	out, _ := cmd.Flags().GetString("out")
	moduleRoot, _ := cmd.Flags().GetString("module-root")
	dry, _ := cmd.Flags().GetBool("dry-run")

	paths, err := gen.Policy(moduleRoot, out, args[0], dry)
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
		fmt.Printf("\nRegister the policy in your module's register.go:\n")
		name := gen.PackageName(args[0])
		fmt.Printf("  gate.RegisterPolicy(%q, &policies.%sPolicy{})\n", name, snakeToPascalCLI(name))
	}
	return nil
}

func snakeToPascalCLI(s string) string {
	parts := strings.Split(s, "_")
	var out strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(p)
		out.WriteRune(rune(strings.ToUpper(string(r[0]))[0]))
		if len(r) > 1 {
			out.WriteString(string(r[1:]))
		}
	}
	return out.String()
}
