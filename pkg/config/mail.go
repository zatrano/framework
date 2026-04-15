package config

// Mail holds mail system configuration.
type Mail struct {
	// Driver: "smtp", "log" (default: "log" in dev, "smtp" in prod).
	Driver string `mapstructure:"driver"`

	// FromName is the default sender display name.
	FromName string `mapstructure:"from_name"`
	// FromEmail is the default sender email address.
	FromEmail string `mapstructure:"from_email"`

	// TemplatesDir is the directory containing email templates (default: "views/mails").
	TemplatesDir string `mapstructure:"templates_dir"`

	// SMTP connection settings.
	SMTP SMTPConfig `mapstructure:"smtp"`
}

// SMTPConfig holds SMTP connection parameters.
type SMTPConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	Encryption string `mapstructure:"encryption"` // "tls", "starttls", or ""
}
