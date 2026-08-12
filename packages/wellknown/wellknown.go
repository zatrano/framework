package wellknown

import (
	"strings"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Config holds security.txt / well-known metadata.
type Config struct {
	ContactEmail  string
	ContactURL    string
	Expires       time.Time
	PolicyURL     string
	Canonical     string
	PreferredLang string
}

// Repository serves RFC 9116 security.txt and related well-known files.
type Repository struct {
	cfg Config
}

// New creates a well-known repository.
func New(cfg Config) *Repository {
	if cfg.Expires.IsZero() {
		cfg.Expires = time.Now().UTC().AddDate(1, 0, 0)
	}
	if cfg.PreferredLang == "" {
		cfg.PreferredLang = "en"
	}
	return &Repository{cfg: cfg}
}

// SecurityTxt renders security.txt content.
func (r *Repository) SecurityTxt() string {
	var b strings.Builder
	if r.cfg.ContactEmail != "" {
		b.WriteString("Contact: mailto:" + r.cfg.ContactEmail + "\n")
	}
	if r.cfg.ContactURL != "" {
		b.WriteString("Contact: " + r.cfg.ContactURL + "\n")
	}
	b.WriteString("Expires: " + r.cfg.Expires.UTC().Format(time.RFC3339) + "\n")
	if r.cfg.PreferredLang != "" {
		b.WriteString("Preferred-Languages: " + r.cfg.PreferredLang + "\n")
	}
	if r.cfg.Canonical != "" {
		b.WriteString("Canonical: " + r.cfg.Canonical + "\n")
	}
	if r.cfg.PolicyURL != "" {
		b.WriteString("Policy: " + r.cfg.PolicyURL + "\n")
	}
	return b.String()
}

// SecurityTxtHandler serves /.well-known/security.txt
func (r *Repository) SecurityTxtHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		resp := http.Text(r.SecurityTxt())
		resp.Header("Content-Type", "text/plain; charset=utf-8")
		return resp
	}
}

// ChangePasswordHandler serves /.well-known/change-password redirect target hint.
func (r *Repository) ChangePasswordHandler(loginPath string) routing.HandlerFunc {
	if loginPath == "" {
		loginPath = "/login"
	}
	return func(req *http.Request) *http.Response {
		return http.Redirect(loginPath, 302)
	}
}
