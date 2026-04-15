package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/security"
)

var jwtCmd = &cobra.Command{
	Use:   "jwt",
	Short: "JWT helpers for local testing",
}

var jwtSignCmd = &cobra.Command{
	Use:   "sign",
	Short: "Print a signed HS256 access token using config (or flags)",
	RunE:  runJWTSign,
}

func init() {
	jwtSignCmd.Flags().String("env", "", "environment name; default ENV or dev")
	jwtSignCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	jwtSignCmd.Flags().Bool("no-dotenv", false, "do not load .env")
	jwtSignCmd.Flags().String("sub", "dev-user", "JWT subject (sub claim)")
	jwtSignCmd.Flags().String("secret", "", "override jwt secret (default: config security.jwt_secret)")

	jwtCmd.AddCommand(jwtSignCmd)
	rootCmd.AddCommand(jwtCmd)
}

func runJWTSign(cmd *cobra.Command, _ []string) error {
	env, _ := cmd.Flags().GetString("env")
	dir, _ := cmd.Flags().GetString("config-dir")
	noDot, _ := cmd.Flags().GetBool("no-dotenv")
	sub, _ := cmd.Flags().GetString("sub")
	sec, _ := cmd.Flags().GetString("secret")

	cfg, err := config.Load(config.LoadOptions{
		Env:       env,
		ConfigDir: dir,
		DotEnv:    !noDot,
	})
	if err != nil {
		return err
	}
	if sec != "" {
		cfg.Security.JWTSecret = sec
	}
	tok, err := security.SignAccessToken(cfg, sub, nil)
	if err != nil {
		return fmt.Errorf("%w\n\nhint: set SECURITY_JWT_SECRET or security.jwt_secret in YAML", err)
	}
	fmt.Fprintln(os.Stdout, tok)
	return nil
}
