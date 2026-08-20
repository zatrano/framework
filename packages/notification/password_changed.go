package notification

// PasswordChangedNotification alerts the user that their password was changed.
type PasswordChangedNotification struct {
	Base
}

// Via uses the mail channel.
func (PasswordChangedNotification) Via() []string {
	return []string{"mail"}
}

// ToMail builds the password-changed notice.
func (PasswordChangedNotification) ToMail(notifiable Notifiable) *MailMessage {
	email := notifiable.RouteNotificationFor("mail")
	body := "Your password was changed. If you did not make this change, reset your password immediately."
	msg := &MailMessage{
		Subject: "Password Changed",
		Text:    body,
		HTML:    "<p>" + body + "</p>",
	}
	if email != "" {
		msg.To = []string{email}
	}
	return msg
}
