package gen

import (
	"fmt"
	"os"
	"path/filepath"
)

// Mail generates a mailable struct + email template files.
// Creates:
//   - baseDir/mails/<name>_mail.go   (Mailable struct)
//   - viewsDir/<name>.html            (email template)
func Mail(moduleRoot, baseDir, viewsDir, rawName string, dryRun bool) ([]string, error) {
	name := normalizeName(rawName)
	if name == "" {
		return nil, fmt.Errorf("invalid mail name %q (use letters, digits, _ or -)", rawName)
	}
	modPath, err := ModuleImportPath(moduleRoot)
	if err != nil {
		return nil, err
	}
	pascal := snakeToPascal(name)

	mailDir := filepath.Join(baseDir, "mails")
	structFile := name + "_mail.go"
	structBody := tmplMailable(name, pascal, modPath)

	tmplFile := name + ".html"
	tmplBody := tmplMailView(name, pascal)

	var written []string

	// Mailable struct
	structPath := filepath.Join(mailDir, structFile)
	written = append(written, structPath)

	// Template view
	tmplPath := filepath.Join(viewsDir, tmplFile)
	written = append(written, tmplPath)

	if dryRun {
		return written, nil
	}

	if err := os.MkdirAll(mailDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(viewsDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(structPath, []byte(structBody), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(tmplPath, []byte(tmplBody), 0o644); err != nil {
		return nil, err
	}
	return written, nil
}

func tmplMailable(pkg, pascal, modImport string) string {
	return fmt.Sprintf(`package mails

import (
	"github.com/zatrano/framework/pkg/mail"
)

// %[2]sMail is a mailable that sends the %[1]s email.
// Send it with:
//
//	app.Mail.SendMailable(ctx, &mails.%[2]sMail{
//	    Name:  "Alice",
//	    Email: "alice@example.com",
//	})
type %[2]sMail struct {
	// Add recipient-specific fields here.
	Name  string
	Email string
}

// Build constructs the email message using the fluent builder API.
func (m *%[2]sMail) Build(b *mail.MessageBuilder) error {
	b.To(m.Name, m.Email).
		Subject("%[2]s Notification").
		View("%[1]s", "default", map[string]any{
			"Name": m.Name,
		})
	return nil
}
`, pkg, pascal, modImport)
}

func tmplMailView(pkg, pascal string) string {
	return fmt.Sprintf(`{{/* %[2]s email template */}}
<h1>Hello, {{ .Name }}!</h1>
<p>This is the <strong>%[1]s</strong> email template.</p>
<p>Edit this file at <code>views/mails/%[1]s.html</code></p>

{{/* 
  This template is rendered inside a layout (e.g. "default").
  The layout file is at views/mails/layouts/default.html
  and injects this content via {{ .Content }}
*/}}
`, pkg, pascal)
}
