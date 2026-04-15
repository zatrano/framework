package cli

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/mail"
)

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Mail management commands",
}

var mailPreviewCmd = &cobra.Command{
	Use:   "preview [template]",
	Short: "Preview an email template in the browser",
	Long: `Starts a local HTTP server that renders email templates for preview.

Without arguments, lists all available templates.
With a template name, renders it with sample data.

Examples:
  zatrano mail preview              # list templates
  zatrano mail preview welcome      # preview welcome template
  zatrano mail preview welcome --port 3001
  
Integrates with Mailpit / MailHog for local development.`,
	RunE: runMailPreview,
}

func init() {
	mailPreviewCmd.Flags().Int("port", 3000, "preview server port")
	mailPreviewCmd.Flags().String("layout", "default", "layout to use for rendering")
	mailPreviewCmd.Flags().String("views", "views/mails", "email templates directory")
	mailPreviewCmd.Flags().String("env", "", "environment profile")
	mailPreviewCmd.Flags().String("config-dir", "config", "config directory")

	mailCmd.AddCommand(mailPreviewCmd)
	rootCmd.AddCommand(mailCmd)
}

func runMailPreview(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	layout, _ := cmd.Flags().GetString("layout")
	viewsDir, _ := cmd.Flags().GetString("views")

	renderer := mail.NewTemplateRenderer(viewsDir, nil)

	addr := fmt.Sprintf(":%d", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || path == "favicon.ico" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>ZATRANO Mail Preview</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; max-width: 800px; margin: 40px auto; padding: 20px; }
h1 { color: #333; }
a { color: #0066cc; text-decoration: none; display: block; padding: 8px 0; }
a:hover { text-decoration: underline; }
.hint { color: #666; font-size: 14px; margin-top: 20px; }
</style>
</head><body>
<h1>📧 ZATRANO Mail Preview</h1>
<p>Navigate to <code>/{template_name}</code> to preview an email template.</p>
<p class="hint">Templates are loaded from <code>%s</code></p>
<p class="hint">Layout: <code>%s</code></p>
</body></html>`, viewsDir, layout)
			return
		}

		// Sample data for preview.
		sampleData := map[string]any{
			"Name":    "John Doe",
			"Email":   "john@example.com",
			"AppName": "ZATRANO",
			"URL":     "https://example.com",
		}

		html, err := renderer.Render(context.Background(), path, layout, sampleData)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintf(w, `<h2>Template Error</h2><pre>%s</pre>
<p><a href="/">← Back</a></p>`, err.Error())
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})

	fmt.Printf("📧 Mail preview server running at http://localhost%s\n", addr)
	fmt.Printf("   Templates: %s\n", viewsDir)
	fmt.Printf("   Layout: %s\n", layout)
	fmt.Println("   Press Ctrl+C to stop")

	return http.ListenAndServe(addr, mux)
}

// loadMailConfig loads mail configuration for CLI commands.
func loadMailConfig(cmd *cobra.Command) (*config.Config, error) {
	envName, _ := cmd.Flags().GetString("env")
	cfgDir, _ := cmd.Flags().GetString("config-dir")
	return config.Load(config.LoadOptions{
		Env:       envName,
		ConfigDir: cfgDir,
		DotEnv:    true,
	})
}
