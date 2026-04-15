package cli

import (
	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/zatrano"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server (Fiber)",
	Long: `Loads configuration from .env (optional), config/{env}.yaml, and environment variables,
then starts the HTTP server. Use --env to override the environment name.`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().String("addr", "", "override HTTP listen address (default: config http_addr)")
	serveCmd.Flags().String("env", "", "environment name (e.g. dev, prod); default ENV or dev")
	serveCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	serveCmd.Flags().Bool("no-dotenv", false, "do not load .env from the working directory")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, _ []string) error {
	envFlag, _ := cmd.Flags().GetString("env")
	configDir, _ := cmd.Flags().GetString("config-dir")
	noDotenv, _ := cmd.Flags().GetBool("no-dotenv")
	addrFlag, _ := cmd.Flags().GetString("addr")

	return zatrano.Start(zatrano.StartOptions{
		Env:       envFlag,
		ConfigDir: configDir,
		Addr:      addrFlag,
		NoDotenv:  noDotenv,
	})
}
