package mail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"
)

// buildMIME constructs a raw MIME email from a Message.
func buildMIME(msg *Message) ([]byte, error) {
	var buf bytes.Buffer

	// Headers.
	buf.WriteString("From: " + msg.From.String() + "\r\n")
	if len(msg.To) > 0 {
		buf.WriteString("To: " + joinAddrs(msg.To) + "\r\n")
	}
	if len(msg.CC) > 0 {
		buf.WriteString("Cc: " + joinAddrs(msg.CC) + "\r\n")
	}
	if msg.ReplyTo.Email != "" {
		buf.WriteString("Reply-To: " + msg.ReplyTo.String() + "\r\n")
	}
	buf.WriteString("Subject: " + msg.Subject + "\r\n")
	buf.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	for k, v := range msg.Headers {
		buf.WriteString(k + ": " + v + "\r\n")
	}

	hasAttachments := len(msg.Attachments) > 0
	hasHTML := msg.HTMLBody != ""
	hasText := msg.TextBody != ""

	if hasAttachments {
		// Mixed multipart (text/html + attachments).
		w := multipart.NewWriter(&buf)
		buf.WriteString("Content-Type: multipart/mixed; boundary=" + w.Boundary() + "\r\n\r\n")

		if hasHTML || hasText {
			if hasHTML && hasText {
				// Alternative part inside mixed.
				altW := multipart.NewWriter(&buf)
				altH := textproto.MIMEHeader{}
				altH.Set("Content-Type", "multipart/alternative; boundary="+altW.Boundary())
				altPart, _ := w.CreatePart(altH)
				writeAlternative(altPart, altW, msg.TextBody, msg.HTMLBody)
			} else if hasHTML {
				h := textproto.MIMEHeader{}
				h.Set("Content-Type", "text/html; charset=utf-8")
				h.Set("Content-Transfer-Encoding", "quoted-printable")
				part, _ := w.CreatePart(h)
				_, _ = part.Write([]byte(msg.HTMLBody))
			} else {
				h := textproto.MIMEHeader{}
				h.Set("Content-Type", "text/plain; charset=utf-8")
				part, _ := w.CreatePart(h)
				_, _ = part.Write([]byte(msg.TextBody))
			}
		}

		for _, att := range msg.Attachments {
			if err := writeAttachment(w, att); err != nil {
				return nil, err
			}
		}
		_ = w.Close()
	} else if hasHTML && hasText {
		// Alternative multipart.
		w := multipart.NewWriter(&buf)
		buf.WriteString("Content-Type: multipart/alternative; boundary=" + w.Boundary() + "\r\n\r\n")
		writeAlternative(&buf, w, msg.TextBody, msg.HTMLBody)
		_ = w.Close()
	} else if hasHTML {
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
		buf.WriteString(msg.HTMLBody)
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		buf.WriteString(msg.TextBody)
	}

	return buf.Bytes(), nil
}

func joinAddrs(addrs []Address) string {
	parts := make([]string, len(addrs))
	for i, a := range addrs {
		parts[i] = a.String()
	}
	return strings.Join(parts, ", ")
}

func writeAlternative(w io.Writer, mw *multipart.Writer, text, html string) {
	th := textproto.MIMEHeader{}
	th.Set("Content-Type", "text/plain; charset=utf-8")
	tp, _ := mw.CreatePart(th)
	_, _ = tp.Write([]byte(text))

	hh := textproto.MIMEHeader{}
	hh.Set("Content-Type", "text/html; charset=utf-8")
	hp, _ := mw.CreatePart(hh)
	_, _ = hp.Write([]byte(html))
}

func writeAttachment(w *multipart.Writer, att Attachment) error {
	ct := att.ContentType
	if ct == "" {
		ct = mime.TypeByExtension(filepath.Ext(att.Filename))
		if ct == "" {
			ct = "application/octet-stream"
		}
	}

	h := textproto.MIMEHeader{}
	h.Set("Content-Type", ct+"; name=\""+att.Filename+"\"")
	h.Set("Content-Transfer-Encoding", "base64")
	if att.Inline {
		h.Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", att.Filename))
	} else {
		h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.Filename))
	}

	part, err := w.CreatePart(h)
	if err != nil {
		return fmt.Errorf("mail: create attachment part: %w", err)
	}

	var data []byte
	if att.Content != nil {
		data = att.Content
	} else if att.Reader != nil {
		data, err = io.ReadAll(att.Reader)
		if err != nil {
			return fmt.Errorf("mail: read attachment: %w", err)
		}
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	_, err = part.Write([]byte(encoded))
	return err
}
