package notification

import "html"

// VerifyEmailNotification delivers a signed email-verification link.
type VerifyEmailNotification struct {
	Base
	VerifyURL string
	AppName   string
	Locale    string
}

// Via uses the mail channel.
func (VerifyEmailNotification) Via() []string {
	return []string{"mail"}
}

// ToMail builds the localized verification email.
func (n VerifyEmailNotification) ToMail(notifiable Notifiable) *MailMessage {
	email := notifiable.RouteNotificationFor("mail")
	repl := map[string]string{
		"app":  resolveMailAppName(n.AppName),
		"link": n.VerifyURL,
	}
	subject := mailTranslate(n.Locale, "auth.mail_verify_subject", repl)
	intro := mailTranslate(n.Locale, "auth.mail_verify_intro", repl)
	action := mailTranslate(n.Locale, "auth.mail_verify_action", repl)
	text := mailTranslate(n.Locale, "auth.mail_verify_text", repl)
	safeLink := html.EscapeString(n.VerifyURL)
	htmlBody := "<p>" + html.EscapeString(intro) + `</p><p><a href="` + safeLink + `">` + html.EscapeString(action) + "</a></p>"
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
