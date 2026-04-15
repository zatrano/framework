package mail

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/zatrano/framework/pkg/i18n"
	"github.com/zatrano/framework/pkg/queue"
)

// MailConfig holds mail system configuration.
type MailConfig struct {
	// Driver: "smtp", "mailgun", "ses", "log" (default: "smtp").
	Driver string `mapstructure:"driver"`

	// From is the default sender address.
	FromName  string `mapstructure:"from_name"`
	FromEmail string `mapstructure:"from_email"`

	// TemplatesDir is the path to email templates (default: "views/mails").
	TemplatesDir string `mapstructure:"templates_dir"`

	// SMTP settings.
	SMTP SMTPConfig `mapstructure:"smtp"`
}

// Manager is the high-level mail facade.
type Manager struct {
	driver   Driver
	renderer *TemplateRenderer
	config   MailConfig
	logger   *zap.Logger
	queue    *queue.Manager // optional, for async sending
}

// New creates a mail manager.
func New(driver Driver, cfg MailConfig, logger *zap.Logger, bundle *i18n.Bundle) *Manager {
	tplDir := cfg.TemplatesDir
	if tplDir == "" {
		tplDir = "views/mails"
	}

	return &Manager{
		driver:   driver,
		renderer: NewTemplateRenderer(tplDir, bundle),
		config:   cfg,
		logger:   logger,
	}
}

// SetQueue enables asynchronous mail sending via the queue system.
func (m *Manager) SetQueue(q *queue.Manager) {
	m.queue = q
}

// Driver returns the underlying mail driver.
func (m *Manager) Driver() Driver { return m.driver }

// Renderer returns the template renderer.
func (m *Manager) Renderer() *TemplateRenderer { return m.renderer }

// ─── Sending ───────────────────────────────────────────────────────────────

// Send sends a message synchronously.
func (m *Manager) Send(ctx context.Context, msg *Message) error {
	m.applyDefaults(msg)
	if err := m.driver.Send(ctx, msg); err != nil {
		m.logger.Error("mail send failed",
			zap.String("driver", m.driver.Name()),
			zap.String("to", joinAddrs(msg.To)),
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return fmt.Errorf("mail: send: %w", err)
	}
	m.logger.Info("mail sent",
		zap.String("driver", m.driver.Name()),
		zap.String("to", joinAddrs(msg.To)),
		zap.String("subject", msg.Subject),
	)
	return nil
}

// Queue dispatches a mail job to the background queue for async sending.
// Falls back to synchronous Send if no queue is configured.
func (m *Manager) Queue(ctx context.Context, msg *Message) error {
	m.applyDefaults(msg)
	if m.queue == nil {
		return m.Send(ctx, msg)
	}
	job := &MailJob{
		Msg: *msg,
	}
	return m.queue.Dispatch(ctx, job)
}

// SendMailable builds and sends a Mailable synchronously.
func (m *Manager) SendMailable(ctx context.Context, mailable Mailable) error {
	msg, err := m.buildMailable(ctx, mailable)
	if err != nil {
		return err
	}
	return m.Send(ctx, msg)
}

// QueueMailable builds and dispatches a Mailable via the queue.
func (m *Manager) QueueMailable(ctx context.Context, mailable Mailable) error {
	msg, err := m.buildMailable(ctx, mailable)
	if err != nil {
		return err
	}
	return m.Queue(ctx, msg)
}

// ─── Template Rendering ────────────────────────────────────────────────────

// SendTemplate renders a template and sends it.
func (m *Manager) SendTemplate(ctx context.Context, to []Address, subject, tmplName, layout string, data map[string]any) error {
	html, err := m.renderer.Render(ctx, tmplName, layout, data)
	if err != nil {
		return err
	}
	msg := &Message{
		To:       to,
		Subject:  subject,
		HTMLBody: html,
	}
	return m.Send(ctx, msg)
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func (m *Manager) applyDefaults(msg *Message) {
	if msg.From.Email == "" {
		msg.From = Address{
			Name:  m.config.FromName,
			Email: m.config.FromEmail,
		}
	}
}

func (m *Manager) buildMailable(ctx context.Context, mailable Mailable) (*Message, error) {
	b := &MessageBuilder{
		msg:      &Message{},
		renderer: m.renderer,
		ctx:      ctx,
	}
	if err := mailable.Build(b); err != nil {
		return nil, fmt.Errorf("mail: build mailable: %w", err)
	}
	return b.msg, nil
}

// ─── MessageBuilder ────────────────────────────────────────────────────────

// MessageBuilder provides a fluent API for constructing messages in Mailable.Build().
type MessageBuilder struct {
	msg      *Message
	renderer *TemplateRenderer
	ctx      context.Context
}

func (b *MessageBuilder) From(name, email string) *MessageBuilder {
	b.msg.From = Address{Name: name, Email: email}
	return b
}

func (b *MessageBuilder) To(name, email string) *MessageBuilder {
	b.msg.To = append(b.msg.To, Address{Name: name, Email: email})
	return b
}

func (b *MessageBuilder) CC(name, email string) *MessageBuilder {
	b.msg.CC = append(b.msg.CC, Address{Name: name, Email: email})
	return b
}

func (b *MessageBuilder) BCC(name, email string) *MessageBuilder {
	b.msg.BCC = append(b.msg.BCC, Address{Name: name, Email: email})
	return b
}

func (b *MessageBuilder) ReplyTo(name, email string) *MessageBuilder {
	b.msg.ReplyTo = Address{Name: name, Email: email}
	return b
}

func (b *MessageBuilder) Subject(s string) *MessageBuilder {
	b.msg.Subject = s
	return b
}

func (b *MessageBuilder) Text(body string) *MessageBuilder {
	b.msg.TextBody = body
	return b
}

func (b *MessageBuilder) HTML(body string) *MessageBuilder {
	b.msg.HTMLBody = body
	return b
}

// View renders an HTML template and sets it as the email body.
func (b *MessageBuilder) View(name, layout string, data map[string]any) *MessageBuilder {
	html, err := b.renderer.Render(b.ctx, name, layout, data)
	if err == nil {
		b.msg.HTMLBody = html
	}
	return b
}

// Attach adds a file attachment by path.
func (b *MessageBuilder) Attach(filePath string) *MessageBuilder {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return b
	}
	b.msg.Attachments = append(b.msg.Attachments, Attachment{
		Filename: filePath,
		Content:  data,
	})
	return b
}

// AttachData adds an attachment from raw bytes.
func (b *MessageBuilder) AttachData(filename string, data []byte, contentType string) *MessageBuilder {
	b.msg.Attachments = append(b.msg.Attachments, Attachment{
		Filename:    filename,
		Content:     data,
		ContentType: contentType,
	})
	return b
}

func (b *MessageBuilder) Header(key, value string) *MessageBuilder {
	if b.msg.Headers == nil {
		b.msg.Headers = make(map[string]string)
	}
	b.msg.Headers[key] = value
	return b
}

// ─── MailJob ───────────────────────────────────────────────────────────────

// MailJob is a queue job that sends an email asynchronously.
type MailJob struct {
	queue.BaseJob
	Msg Message `json:"msg"`
}

func (j *MailJob) Name() string            { return "zatrano_mail" }
func (j *MailJob) Queue() string           { return "mails" }
func (j *MailJob) Retries() int            { return 3 }
func (j *MailJob) Timeout() time.Duration  { return 30 * time.Second }

func (j *MailJob) Handle(ctx context.Context) error {
	// This is handled by the MailWorkerMiddleware — see RegisterMailJob.
	// The worker resolves the Manager and calls Send.
	return fmt.Errorf("mail job requires MailWorkerMiddleware — register with mail.RegisterMailJob()")
}

// RegisterMailJob registers the mail job with the queue manager and provides
// a custom handler that uses the mail Manager to send.
func RegisterMailJob(qm *queue.Manager, mm *Manager) {
	qm.Register("zatrano_mail", func() queue.Job {
		return &MailJobWithManager{manager: mm}
	})
}

// MailJobWithManager wraps MailJob with a reference to the mail Manager.
type MailJobWithManager struct {
	queue.BaseJob
	Msg     Message `json:"msg"`
	manager *Manager
}

func (j *MailJobWithManager) Name() string            { return "zatrano_mail" }
func (j *MailJobWithManager) Queue() string           { return "mails" }
func (j *MailJobWithManager) Retries() int            { return 3 }
func (j *MailJobWithManager) Timeout() time.Duration  { return 30 * time.Second }

func (j *MailJobWithManager) Handle(ctx context.Context) error {
	return j.manager.Send(ctx, &j.Msg)
}
