package notification

import (
	"fmt"
	"net/url"
)

// PasswordResetNotification delivers a password-reset link over the mail channel.
type PasswordResetNotification struct {
	Base
	Token    string
	ResetURL string
}

// Via uses the mail channel.
func (PasswordResetNotification) Via() []string {
	return []string{"mail"}
}

// ToMail builds the reset email.
func (n PasswordResetNotification) ToMail(notifiable Notifiable) *MailMessage {
	email := notifiable.RouteNotificationFor("mail")
	link := fmt.Sprintf("%s?email=%s&token=%s", n.ResetURL, url.QueryEscape(email), url.QueryEscape(n.Token))
	body := fmt.Sprintf("Reset your password using token %s\n\n%s", n.Token, link)
	msg := &MailMessage{
		Subject: "Reset Password",
		Text:    body,
		HTML:    "<p>Reset your password using token " + n.Token + "</p><p><a href=\"" + link + "\">Reset Password</a></p>",
	}
	if email != "" {
		msg.To = []string{email}
	}
	return msg
}
