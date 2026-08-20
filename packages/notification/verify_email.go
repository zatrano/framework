package notification

// VerifyEmailNotification delivers a signed email-verification link.
type VerifyEmailNotification struct {
	Base
	VerifyURL string
}

// Via uses the mail channel.
func (VerifyEmailNotification) Via() []string {
	return []string{"mail"}
}

// ToMail builds the verification email.
func (n VerifyEmailNotification) ToMail(notifiable Notifiable) *MailMessage {
	email := notifiable.RouteNotificationFor("mail")
	body := "Please verify your email address:\n\n" + n.VerifyURL
	msg := &MailMessage{
		Subject: "Verify Email Address",
		Text:    body,
		HTML:    `<p>Please verify your email address:</p><p><a href="` + n.VerifyURL + `">Verify Email</a></p>`,
	}
	if email != "" {
		msg.To = []string{email}
	}
	return msg
}
