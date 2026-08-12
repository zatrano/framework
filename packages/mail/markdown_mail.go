package mail

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/packages/markdown"
	"github.com/zatrano/framework/packages/view"
)

// Mailable is a renderable email.
type Mailable interface {
	Envelope() Envelope
	Content() Content
}

// Envelope holds addressing metadata.
type Envelope struct {
	To      []string
	Cc      []string
	Bcc     []string
	Subject string
	From    string
}

// Content describes email body sources.
type Content struct {
	HTML     string
	Text     string
	Markdown string
	View     string
	Data     map[string]any
}

// SetView attaches a view engine for template mails.
func (m *Manager) SetView(engine *view.Engine) {
	m.view = engine
}

// SendMarkdownView loads a markdown mail template, substitutes {{ $vars }}, and sends it.
func (m *Manager) SendMarkdownView(address, subject, viewName string, data map[string]any) error {
	if m.view == nil {
		return fmt.Errorf("mail view engine is not configured")
	}
	path := filepath.Join(m.view.Directory(), strings.ReplaceAll(viewName, ".", string(os.PathSeparator))+".md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("mail markdown view [%s] not found", viewName)
	}
	body := string(raw)
	for key, value := range data {
		body = strings.ReplaceAll(body, "{{ $"+key+" }}", fmt.Sprint(value))
		body = strings.ReplaceAll(body, "{{$"+key+"}}", fmt.Sprint(value))
	}
	return m.SendMarkdown(address, subject, body)
}

// SendMarkdown sends a markdown email (HTML + text fallback).
func (m *Manager) SendMarkdown(address, subject, md string) error {
	return m.Send(&Message{
		To:      []string{address},
		Subject: subject,
		HTML:    wrapMailHTML(markdown.ToHTML(md)),
		Text:    markdown.ToText(md),
	})
}

// SendView renders a view and sends it as HTML mail.
func (m *Manager) SendView(address, subject, viewName string, data map[string]any) error {
	if m.view == nil {
		return fmt.Errorf("mail view engine is not configured")
	}
	htmlBody, err := m.view.Render(viewName, data)
	if err != nil {
		return err
	}
	return m.Send(&Message{
		To:      []string{address},
		Subject: subject,
		HTML:    htmlBody,
		Text:    stripTags(htmlBody),
	})
}

// SendMailable sends a structured mailable.
func (m *Manager) SendMailable(mailable Mailable) error {
	envelope := mailable.Envelope()
	content := mailable.Content()

	message := &Message{
		From:    envelope.From,
		To:      envelope.To,
		Cc:      envelope.Cc,
		Bcc:     envelope.Bcc,
		Subject: envelope.Subject,
		HTML:    content.HTML,
		Text:    content.Text,
	}

	switch {
	case content.Markdown != "":
		message.HTML = wrapMailHTML(markdown.ToHTML(content.Markdown))
		if message.Text == "" {
			message.Text = markdown.ToText(content.Markdown)
		}
	case content.View != "":
		if m.view == nil {
			return fmt.Errorf("mail view engine is not configured")
		}
		htmlBody, err := m.view.Render(content.View, content.Data)
		if err != nil {
			return err
		}
		message.HTML = htmlBody
		if message.Text == "" {
			message.Text = stripTags(htmlBody)
		}
	}

	return m.Send(message)
}

func wrapMailHTML(body string) string {
	return `<!DOCTYPE html><html><body style="font-family:Georgia,serif;line-height:1.5;color:#1b2433;">` +
		body +
		`</body></html>`
}

func stripTags(input string) string {
	out := input
	out = strings.ReplaceAll(out, "<br>", "\n")
	out = strings.ReplaceAll(out, "<br/>", "\n")
	out = strings.ReplaceAll(out, "<br />", "\n")
	out = strings.ReplaceAll(out, "</p>", "\n\n")
	out = strings.ReplaceAll(out, "</h1>", "\n")
	out = strings.ReplaceAll(out, "</h2>", "\n")
	out = strings.ReplaceAll(out, "</li>", "\n")
	var b strings.Builder
	inTag := false
	for _, r := range out {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
