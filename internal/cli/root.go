package cli

import (
	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/meta"
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "zatrano",
	Short: "ZATRANO framework CLI",
	Long: `ZATRANO is a modular Go web framework for APIs and server-rendered forms.

Stack: Fiber v3, PostgreSQL, Redis, GORM.

Typical flow: copy config/examples → zatrano config validate → zatrano doctor → zatrano serve.
CI: zatrano config validate -q && zatrano openapi validate --merged && zatrano verify`,
	Version: meta.Version,
	Example: `  zatrano serve
  zatrano config validate
  zatrano doctor
  zatrano routes
  zatrano config print --paths-only
  zatrano openapi validate --merged
  zatrano verify
  zatrano verify --race
  zatrano openapi export --output api/openapi.merged.yaml
  zatrano gen module my_feature
  zatrano gen crud my_feature
  zatrano gen request my_feature
  zatrano gen wire my_feature
  zatrano db migrate`,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}
