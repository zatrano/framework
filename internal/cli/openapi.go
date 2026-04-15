package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/openapi"
)

var openapiCmd = &cobra.Command{
	Use:   "openapi",
	Short: "OpenAPI utilities",
}

var openapiValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate an OpenAPI 3.x document (YAML or JSON)",
	Long: `Without --merged: validates a single file (--path or positional argument).

With --merged: validates the same combined document as live /openapi.yaml and zatrano openapi export
(base OpenAPI + built-in framework operations). Optional positional argument overrides --base.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runOpenAPIValidate,
}

var openapiExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Write merged OpenAPI (repo file + built-in framework routes) to a file",
	Long: `Loads the base spec (if missing, starts from a minimal document), applies MergeFrameworkRoutes,
then writes YAML suitable for CI or docs publishing.`,
	RunE: runOpenAPIExport,
}

func init() {
	openapiValidateCmd.Flags().String("path", "api/openapi.yaml", "path to openapi spec (ignored when --merged)")
	openapiValidateCmd.Flags().Bool("merged", false, "validate base file merged with built-in framework routes (like /openapi.yaml)")
	openapiValidateCmd.Flags().String("base", "api/openapi.yaml", "base OpenAPI path for --merged (overridden by positional arg)")
	openapiExportCmd.Flags().String("base", "api/openapi.yaml", "base OpenAPI file to merge (optional)")
	openapiExportCmd.Flags().String("output", "api/openapi.merged.yaml", "output path (- for stdout)")
	openapiCmd.AddCommand(openapiValidateCmd, openapiExportCmd)
	rootCmd.AddCommand(openapiCmd)
}

func runOpenAPIValidate(cmd *cobra.Command, args []string) error {
	merged, _ := cmd.Flags().GetBool("merged")
	base, _ := cmd.Flags().GetString("base")
	if merged && len(args) > 0 {
		base = args[0]
	}
	if merged {
		_, doc, err := openapi.LoadMergedDocument(context.Background(), base)
		if err != nil {
			return err
		}
		fmt.Printf("ok: merged OpenAPI (base %q) OpenAPI %s\n", base, doc.OpenAPI)
		return nil
	}

	path, _ := cmd.Flags().GetString("path")
	if len(args) > 0 {
		path = args[0]
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(raw)
	if err != nil {
		return fmt.Errorf("parse spec: %w", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		return fmt.Errorf("invalid OpenAPI: %w", err)
	}
	fmt.Printf("ok: %s (OpenAPI %s)\n", path, doc.OpenAPI)
	return nil
}

func runOpenAPIExport(cmd *cobra.Command, _ []string) error {
	base, _ := cmd.Flags().GetString("base")
	out, _ := cmd.Flags().GetString("output")
	raw, _, err := openapi.LoadMergedDocument(context.Background(), base)
	if err != nil {
		return err
	}
	if out == "-" {
		_, err := os.Stdout.Write(raw)
		return err
	}
	if err := os.WriteFile(out, raw, 0o644); err != nil {
		return err
	}
	fmt.Printf("ok: wrote merged spec to %s\n", out)
	return nil
}
