package notification

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
	"sync"

	"github.com/zatrano/framework/packages/log"
	"github.com/zatrano/framework/packages/view"
)

// MailMessage represents an email message.
type MailMessage struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	ReplyTo     []string
	Subject     string
	HTML        string
	Text        string
	Headers     map[string]string
	Attachments []Attachment
}

// Mailer sends email messages.
type Mailer interface {
	Send(message *MailMessage) error
}

// MailManager resolves mailers (email transport used by the mail notification channel).
type MailManager struct {
	defaultMailer string
	mailers       map[string]Mailer
	fromAddress   string
	fromName      string
	view          *view.Engine
}

// NewMailManager creates a mail manager.
func NewMailManager(defaultMailer, fromAddress, fromName string, mailers map[string]Mailer) *MailManager {
	return &MailManager{
		defaultMailer: defaultMailer,
		mailers:       mailers,
		fromAddress:   fromAddress,
		fromName:      fromName,
	}
}

// Mailer returns a named mailer.
func (m *MailManager) Mailer(name ...string) Mailer {
	mailer := m.defaultMailer
	if len(name) > 0 && name[0] != "" {
		mailer = name[0]
	}
	return m.mailers[mailer]
}

// Send sends a message using the default mailer.
func (m *MailManager) Send(message *MailMessage) error {
	if message.From == "" {
		if m.fromName != "" {
			message.From = fmt.Sprintf("%s <%s>", m.fromName, m.fromAddress)
		} else {
			message.From = m.fromAddress
		}
	}
	return m.Mailer().Send(message)
}

// LogMailer writes emails to the logger.
type LogMailer struct {
	logger *log.Logger
	mu     sync.Mutex
}

// NewLogMailer creates a log mailer.
func NewLogMailer(logger *log.Logger) *LogMailer {
	return &LogMailer{logger: logger}
}

// Send logs the email.
func (m *LogMailer) Send(message *MailMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logger.Infof("mail to=%v reply_to=%v subject=%q attachments=%d body=%q",
		message.To, message.ReplyTo, message.Subject, len(message.Attachments), firstBody(message))
	return nil
}

// SMTPConfig holds SMTP settings.
type SMTPConfig struct {
	Host       string
	Port       string
	Username   string
	Password   string
	// Encryption selects the TLS mode:
	//   "" / "tls" / "starttls" = plain TCP then STARTTLS (typical port 587)
	//   "ssl" = implicit TLS / SMTPS (typical port 465)
	Encryption string
}

// SMTPMailer sends mail over SMTP.
type SMTPMailer struct {
	config SMTPConfig
}

// NewSMTPMailer creates an SMTP mailer.
func NewSMTPMailer(config SMTPConfig) *SMTPMailer {
	return &SMTPMailer{config: config}
}

// Send delivers the message through SMTP.
func (m *SMTPMailer) Send(message *MailMessage) error {
	addr := m.config.Host + ":" + m.config.Port
	recipients := append(append([]string{}, message.To...), message.Cc...)
	recipients = append(recipients, message.Bcc...)
	from := extractAddress(message.From)
	payload := buildMIME(message)

	if useImplicitTLS(m.config) {
		return sendSMTPTLS(m.config, addr, from, recipients, payload)
	}
	return sendSMTPPlain(m.config, addr, from, recipients, payload)
}

func useImplicitTLS(cfg SMTPConfig) bool {
	enc := strings.ToLower(strings.TrimSpace(cfg.Encryption))
	if enc == "ssl" {
		return true
	}
	return strings.TrimSpace(cfg.Port) == "465"
}

func sendSMTPPlain(cfg SMTPConfig, addr, from string, recipients []string, payload []byte) error {
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	return smtp.SendMail(addr, auth, from, recipients, payload)
}

func sendSMTPTLS(cfg SMTPConfig, addr, from string, recipients []string, payload []byte) error {
	tlsCfg := &tls.Config{
		ServerName: cfg.Host,
		MinVersion: tls.VersionTLS12,
	}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// BuildMIME renders a message as raw MIME bytes.
func BuildMIME(message *MailMessage) []byte {
	return buildMIME(message)
}

func firstBody(message *MailMessage) string {
	if message.Text != "" {
		return message.Text
	}
	return message.HTML
}

func extractAddress(from string) string {
	if start := strings.Index(from, "<"); start >= 0 {
		end := strings.Index(from, ">")
		if end > start {
			return from[start+1 : end]
		}
	}
	return from
}

func buildMIME(message *MailMessage) []byte {
	var b strings.Builder
	b.WriteString("From: " + message.From + "\r\n")
	b.WriteString("To: " + strings.Join(message.To, ", ") + "\r\n")
	if len(message.Cc) > 0 {
		b.WriteString("Cc: " + strings.Join(message.Cc, ", ") + "\r\n")
	}
	if len(message.ReplyTo) > 0 {
		b.WriteString("Reply-To: " + strings.Join(message.ReplyTo, ", ") + "\r\n")
	}
	b.WriteString("Subject: " + message.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	for key, value := range message.Headers {
		b.WriteString(key + ": " + value + "\r\n")
	}

	if len(message.Attachments) == 0 {
		writeBody(&b, message)
		return []byte(b.String())
	}

	mixed := "zatrano-mixed"
	b.WriteString("Content-Type: multipart/mixed; boundary=" + mixed + "\r\n\r\n")
	b.WriteString("--" + mixed + "\r\n")
	writeBody(&b, message)
	for _, att := range message.Attachments {
		b.WriteString("\r\n--" + mixed + "\r\n")
		writeAttachment(&b, att)
	}
	b.WriteString("\r\n--" + mixed + "--")
	return []byte(b.String())
}

func writeBody(b *strings.Builder, message *MailMessage) {
	if message.HTML != "" && message.Text != "" {
		boundary := "zatrano-boundary"
		b.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n\r\n")
		b.WriteString("--" + boundary + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(message.Text + "\r\n")
		b.WriteString("--" + boundary + "\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(message.HTML + "\r\n")
		b.WriteString("--" + boundary + "--")
		return
	}
	if message.HTML != "" {
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(message.HTML)
		return
	}
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(message.Text)
}

func writeAttachment(b *strings.Builder, att Attachment) {
	name := att.Name
	if name == "" {
		name = "attachment.bin"
	}
	ct := att.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	disposition := "attachment"
	if att.Inline {
		disposition = "inline"
	}
	b.WriteString("Content-Type: " + ct + "; name=\"" + name + "\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	b.WriteString("Content-Disposition: " + disposition + "; filename=\"" + name + "\"\r\n")
	if att.Inline && att.ContentID != "" {
		b.WriteString("Content-ID: <" + att.ContentID + ">\r\n")
	}
	b.WriteString("\r\n")
	encoded := base64.StdEncoding.EncodeToString(att.Content)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		b.WriteString(encoded[i:end] + "\r\n")
	}
}
