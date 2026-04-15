package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPConfig holds SMTP connection settings.
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	// Encryption: "tls", "starttls", or "" (none).
	Encryption string `mapstructure:"encryption"`
}

// SMTPDriver sends emails via SMTP (net/smtp).
type SMTPDriver struct {
	config SMTPConfig
}

// NewSMTPDriver creates an SMTP mail driver.
func NewSMTPDriver(cfg SMTPConfig) *SMTPDriver {
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	return &SMTPDriver{config: cfg}
}

// Ensure SMTPDriver implements Driver at compile time.
var _ Driver = (*SMTPDriver)(nil)

func (d *SMTPDriver) Name() string { return "smtp" }

func (d *SMTPDriver) Send(_ context.Context, msg *Message) error {
	addr := fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)

	// Build raw email.
	body, err := buildMIME(msg)
	if err != nil {
		return fmt.Errorf("mail: build mime: %w", err)
	}

	// Collect recipient addresses.
	var recipients []string
	for _, to := range msg.To {
		recipients = append(recipients, to.Email)
	}
	for _, cc := range msg.CC {
		recipients = append(recipients, cc.Email)
	}
	for _, bcc := range msg.BCC {
		recipients = append(recipients, bcc.Email)
	}

	// Auth (if credentials provided).
	var auth smtp.Auth
	if d.config.Username != "" {
		auth = smtp.PlainAuth("", d.config.Username, d.config.Password, d.config.Host)
	}

	switch strings.ToLower(d.config.Encryption) {
	case "tls":
		return d.sendTLS(addr, auth, msg.From.Email, recipients, body)
	default:
		return smtp.SendMail(addr, auth, msg.From.Email, recipients, body)
	}
}

func (d *SMTPDriver) sendTLS(addr string, auth smtp.Auth, from string, to []string, body []byte) error {
	tlsConfig := &tls.Config{ServerName: d.config.Host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("mail: tls dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("mail: smtp client: %w", err)
	}
	defer func() { _ = client.Quit() }()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mail: auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail: mail from: %w", err)
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("mail: rcpt %s: %w", addr, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: data: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("mail: write: %w", err)
	}
	return w.Close()
}
