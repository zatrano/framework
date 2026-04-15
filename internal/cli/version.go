package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/meta"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print ZATRANO CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(meta.Version)
	},
}
