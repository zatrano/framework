package notification

import (
	"path/filepath"
	"strings"
)

// Attachment is a file attached to an email message.
type Attachment struct {
	Name        string
	ContentType string
	Content     []byte
	Inline      bool
	ContentID   string
}

// Attach adds a file attachment to the message.
func (m *MailMessage) Attach(filename string, content []byte, contentType ...string) *MailMessage {
	if m == nil {
		return m
	}
	ct := "application/octet-stream"
	if len(contentType) > 0 && strings.TrimSpace(contentType[0]) != "" {
		ct = contentType[0]
	}
	m.Attachments = append(m.Attachments, Attachment{
		Name:        filepath.Base(filename),
		ContentType: ct,
		Content:     content,
	})
	return m
}

// AttachInline adds an inline attachment with a Content-ID.
func (m *MailMessage) AttachInline(filename string, content []byte, contentID string, contentType ...string) *MailMessage {
	if m == nil {
		return m
	}
	ct := "application/octet-stream"
	if len(contentType) > 0 && strings.TrimSpace(contentType[0]) != "" {
		ct = contentType[0]
	}
	m.Attachments = append(m.Attachments, Attachment{
		Name:        filepath.Base(filename),
		ContentType: ct,
		Content:     content,
		Inline:      true,
		ContentID:   contentID,
	})
	return m
}

// SetReplyTo sets Reply-To addresses.
func (m *MailMessage) SetReplyTo(addresses ...string) *MailMessage {
	if m == nil {
		return m
	}
	m.ReplyTo = append([]string{}, addresses...)
	return m
}
