package notification

import "html"

// PasswordChangedNotification alerts the user that their password was changed.
type PasswordChangedNotification struct {
	Base
	AppName string
	Locale  string
}

// Via uses the mail channel.
func (PasswordChangedNotification) Via() []string {
	return []string{"mail"}
}

// ToMail builds the localized password-changed notice.
func (n PasswordChangedNotification) ToMail(notifiable Notifiable) *MailMessage {
	email := notifiable.RouteNotificationFor("mail")
	repl := map[string]string{
		"app": resolveMailAppName(n.AppName),
	}
	subject := mailTranslate(n.Locale, "auth.mail_password_changed_subject", repl)
	body := mailTranslate(n.Locale, "auth.mail_password_changed_body", repl)
	msg := &MailMessage{
		Subject: subject,
		Text:    body,
		HTML:    "<p>" + html.EscapeString(body) + "</p>",
	}
	if email != "" {
		msg.To = []string{email}
	}
	return msg
}
