package notification

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
)

// PasswordResetNotification delivers a password-reset link over the mail channel.
type PasswordResetNotification struct {
	Base
	Token         string
	ResetURL      string
	ExpireMinutes int    // optional; defaults to 60
	AppName       string // optional override of app.name
	Locale        string // optional override of APP_LOCALE
}

// Via uses the mail channel.
func (PasswordResetNotification) Via() []string {
	return []string{"mail"}
}

// ToMail builds the localized reset email (link-only body; token only in the URL).
func (n PasswordResetNotification) ToMail(notifiable Notifiable) *MailMessage {
	email := notifiable.RouteNotificationFor("mail")
	link := fmt.Sprintf("%s?email=%s&token=%s", n.ResetURL, url.QueryEscape(email), url.QueryEscape(n.Token))
	minutes := n.ExpireMinutes
	if minutes <= 0 {
		minutes = 60
	}
	repl := map[string]string{
		"app":     resolveMailAppName(n.AppName),
		"link":    link,
		"minutes": strconv.Itoa(minutes),
	}
	subject := mailTranslate(n.Locale, "auth.mail_reset_subject", repl)
	intro := mailTranslate(n.Locale, "auth.mail_reset_intro", repl)
	action := mailTranslate(n.Locale, "auth.mail_reset_action", repl)
	expire := mailTranslate(n.Locale, "auth.mail_reset_expire_hint", repl)
	ignore := mailTranslate(n.Locale, "auth.mail_reset_ignore", repl)
	text := mailTranslate(n.Locale, "auth.mail_reset_text", repl)
	safeLink := html.EscapeString(link)
	htmlBody := "<p>" + html.EscapeString(intro) + `</p><p><a href="` + safeLink + `">` + html.EscapeString(action) + "</a></p><p>" +
		html.EscapeString(expire) + "</p><p>" + html.EscapeString(ignore) + "</p>"
	msg := &MailMessage{
		Subject: subject,
		Text:    text,
		HTML:    htmlBody,
	}
	if email != "" {
		msg.To = []string{email}
	}
	return msg
}
