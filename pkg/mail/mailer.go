package mail

import (
	"context"
	"io"
)

// Driver is the backend interface for sending emails.
// Implementations: SMTP, Mailgun, SES.
type Driver interface {
	// Send delivers a fully built message.
	Send(ctx context.Context, msg *Message) error

	// Name returns the driver name (e.g. "smtp", "mailgun", "ses").
	Name() string
}

// Message represents an email to be sent.
type Message struct {
	From        Address   `json:"from"`
	To          []Address `json:"to"`
	CC          []Address `json:"cc,omitempty"`
	BCC         []Address `json:"bcc,omitempty"`
	ReplyTo     Address   `json:"reply_to,omitempty"`
	Subject     string    `json:"subject"`
	TextBody    string    `json:"text_body,omitempty"`
	HTMLBody    string    `json:"html_body,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// Address is an email address with optional display name.
type Address struct {
	Name    string `json:"name,omitempty"`
	Email   string `json:"email"`
}

// String returns "Name <Email>" or just "Email".
func (a Address) String() string {
	if a.Name != "" {
		return a.Name + " <" + a.Email + ">"
	}
	return a.Email
}

// Attachment represents a file attached to an email.
type Attachment struct {
	// Filename is the display name of the attachment.
	Filename string `json:"filename"`

	// ContentType is the MIME type (e.g. "application/pdf"). Auto-detected if empty.
	ContentType string `json:"content_type,omitempty"`

	// Content is the raw file data. Either Content or Reader must be set.
	Content []byte `json:"-"`

	// Reader provides streaming access to the attachment data.
	Reader io.Reader `json:"-"`

	// Inline when true, embeds the attachment inline (Content-Disposition: inline).
	Inline bool `json:"inline,omitempty"`
}

// Mailable is the interface for structured, reusable email definitions.
// Generate stubs with: zatrano gen mail <name>
type Mailable interface {
	// Build constructs the Message. Use the builder helpers on MessageBuilder.
	Build(b *MessageBuilder) error
}
